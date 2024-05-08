/*
Copyright 2023 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notiv1 "github.com/fluxcd/notification-controller/api/v1"
	notiv1beta3 "github.com/fluxcd/notification-controller/api/v1beta3"
	events "github.com/fluxcd/pkg/apis/event/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestNotification(t *testing.T) {
	g := NewWithT(t)

	ctx := context.TODO()
	branchName := "test-notification"
	testID := branchName + "-" + randStringRunes(5)
	defer cfg.notificationCfg.closeChan()

	// Setup Flux resources
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: foobar`
	repoUrl := getTransportURL(cfg.applicationRepository)
	client, err := getRepository(ctx, t.TempDir(), repoUrl, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())
	files := make(map[string]io.Reader)
	files["configmap.yaml"] = strings.NewReader(manifest)
	err = commitAndPushAll(ctx, client, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testID,
		},
	}
	g.Expect(testEnv.Create(ctx, &namespace)).To(Succeed())
	defer testEnv.Delete(ctx, &namespace)

	provider := notiv1beta3.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: testID,
		},
		Spec: notiv1beta3.ProviderSpec{
			Type:    cfg.notificationCfg.providerType,
			Address: cfg.notificationCfg.providerAddress,
			Channel: cfg.notificationCfg.providerChannel,
		},
	}

	if cfg.notificationCfg.secret != nil {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testID,
				Namespace: testID,
			},
			StringData: cfg.notificationCfg.secret,
		}

		g.Expect(testEnv.Create(ctx, &secret)).To(Succeed())
		defer testEnv.Delete(ctx, &secret)

		provider.Spec.SecretRef = &meta.LocalObjectReference{
			Name: testID,
		}
	}

	g.Expect(testEnv.Create(ctx, &provider)).To(Succeed())
	defer testEnv.Delete(ctx, &provider)

	alert := notiv1beta3.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: testID,
		},
		Spec: notiv1beta3.AlertSpec{
			ProviderRef: meta.LocalObjectReference{
				Name: provider.Name,
			},
			EventSources: []notiv1.CrossNamespaceObjectReference{
				{
					Kind:      "Kustomization",
					Name:      testID,
					Namespace: testID,
				},
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &alert)).ToNot(HaveOccurred())
	defer testEnv.Delete(ctx, &alert)

	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.Interval = metav1.Duration{Duration: 30 * time.Second}
		spec.HealthChecks = []meta.NamespacedObjectKindReference{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "foobar",
				Namespace:  testID,
			},
		}
	}
	g.Expect(setUpFluxConfig(ctx, testID, nsConfig{
		repoURL: repoUrl,
		ref: &sourcev1.GitRepositoryRef{
			Branch: branchName,
		},
		path:         "./",
		modifyKsSpec: modifyKsSpec,
	})).To(Succeed())
	t.Cleanup(func() {
		err := tearDownFluxConfig(ctx, testID)
		if err != nil {
			t.Logf("failed to delete resources in '%s' namespace: %s", testID, err)
		}
	})

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv, testID, testID)
		if err != nil {
			t.Log(err)
			return false
		}
		return true
	}, testTimeout, testInterval).Should(BeTrue())

	// Wait to read event from notification channel.
	g.Eventually(func() bool {
		select {
		case eventJson := <-cfg.notificationCfg.notificationChan:
			event := &events.Event{}
			err := json.Unmarshal([]byte(eventJson), event)
			if err != nil {
				t.Logf("the received event type does not match Flux format, error: %v", err)
				return false
			}

			if event.InvolvedObject.Kind == kustomizev1.KustomizationKind &&
				event.InvolvedObject.Name == testID && event.InvolvedObject.Namespace == testID {
				return true
			}

			return false
		default:
			return false
		}
	}, testTimeout, 1*time.Second).Should(BeTrue())
}
