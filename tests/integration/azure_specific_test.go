//go:build azure
// +build azure

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
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	. "github.com/onsi/gomega"
	giturls "github.com/whilp/git-urls"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	notiv1 "github.com/fluxcd/notification-controller/api/v1"
	notiv1beta2 "github.com/fluxcd/notification-controller/api/v1beta2"
	events "github.com/fluxcd/pkg/apis/event/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestEventHubNotification(t *testing.T) {
	g := NewWithT(t)

	ctx := context.TODO()
	branchName := "test-notification"
	testID := branchName + "-" + randStringRunes(5)

	// Start listening to eventhub with latest offset
	// TODO(somtochiama): Make here provider agnostic
	hub, err := eventhub.NewHubFromConnectionString(cfg.notificationURL)
	g.Expect(err).ToNot(HaveOccurred())
	c := make(chan string, 10)
	handler := func(ctx context.Context, event *eventhub.Event) error {
		c <- string(event.Data)
		return nil
	}
	runtimeInfo, err := hub.GetRuntimeInformation(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(runtimeInfo.PartitionIDs)).To(Equal(1))
	listenerHandler, err := hub.Receive(ctx, runtimeInfo.PartitionIDs[0], handler, eventhub.ReceiveWithLatestOffset())
	g.Expect(err).ToNot(HaveOccurred())

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

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: testID,
		},
		StringData: map[string]string{
			"address": cfg.notificationURL,
		},
	}
	g.Expect(testEnv.Create(ctx, &secret)).To(Succeed())
	defer testEnv.Delete(ctx, &secret)

	provider := notiv1beta2.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: testID,
		},
		Spec: notiv1beta2.ProviderSpec{
			Type:    "azureeventhub",
			Address: repoUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: testID,
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &provider)).To(Succeed())
	defer testEnv.Delete(ctx, &provider)

	alert := notiv1beta2.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: testID,
		},
		Spec: notiv1beta2.AlertSpec{
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

	g.Eventually(func() bool {
		nn := types.NamespacedName{Name: alert.Name, Namespace: alert.Namespace}
		alertObj := &notiv1beta2.Alert{}
		err := testEnv.Get(ctx, nn, alertObj)
		if err != nil {
			return false
		}
		if err := checkReadyCondition(alertObj); err != nil {
			t.Log(err)
			return false
		}

		return true
	}, testTimeout, testInterval).Should(BeTrue())

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

	// Wait to read even from event hub
	g.Eventually(func() bool {
		select {
		case eventJson := <-c:
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
	err = listenerHandler.Close(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	err = hub.Close(ctx)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestAzureDevOpsCommitStatus(t *testing.T) {
	g := NewWithT(t)

	ctx := context.TODO()
	branchName := "commit-status"
	testID := branchName + randStringRunes(5)
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: foobar`

	repoUrl := getTransportURL(cfg.applicationRepository)
	tmpDir := t.TempDir()
	c, err := getRepository(ctx, tmpDir, repoUrl, defaultBranch, cfg.defaultAuthOpts)
	g.Expect(err).ToNot(HaveOccurred())
	files := make(map[string]io.Reader)
	files["configmap.yaml"] = strings.NewReader(manifest)
	err = commitAndPushAll(ctx, c, files, branchName)
	g.Expect(err).ToNot(HaveOccurred())

	modifyKsSpec := func(spec *kustomizev1.KustomizationSpec) {
		spec.HealthChecks = []meta.NamespacedObjectKindReference{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "foobar",
				Namespace:  testID,
			},
		}
	}
	err = setUpFluxConfig(ctx, testID, nsConfig{
		ref: &sourcev1.GitRepositoryRef{
			Branch: branchName,
		},
		repoURL:      repoUrl,
		path:         "./",
		modifyKsSpec: modifyKsSpec,
	})
	g.Expect(err).ToNot(HaveOccurred())
	t.Cleanup(func() {
		err := tearDownFluxConfig(ctx, testID)
		if err != nil {
			log.Printf("failed to delete resources in '%s' namespace: %s", testID, err)
		}
	})

	g.Eventually(func() bool {
		err := verifyGitAndKustomization(ctx, testEnv, testID, testID)
		if err != nil {
			return false
		}
		return true
	}, testTimeout, testInterval)

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops-token",
			Namespace: testID,
		},
		StringData: map[string]string{
			"token": cfg.gitPat,
		},
	}
	g.Expect(testEnv.Create(ctx, &secret)).To(Succeed())
	defer testEnv.Delete(ctx, &secret)

	provider := notiv1beta2.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: testID,
		},
		Spec: notiv1beta2.ProviderSpec{
			Type:    "azuredevops",
			Address: repoUrl,
			SecretRef: &meta.LocalObjectReference{
				Name: "azuredevops-token",
			},
		},
	}
	g.Expect(testEnv.Create(ctx, &provider)).To(Succeed())
	defer testEnv.Delete(ctx, &provider)

	alert := notiv1beta2.Alert{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "azuredevops",
			Namespace: testID,
		},
		Spec: notiv1beta2.AlertSpec{
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
	g.Expect(testEnv.Create(ctx, &alert)).To(Succeed())
	defer testEnv.Delete(ctx, &alert)

	url, err := ParseAzureDevopsURL(repoUrl)
	g.Expect(err).ToNot(HaveOccurred())

	rev, err := c.Head()
	g.Expect(err).ToNot(HaveOccurred())

	connection := azuredevops.NewPatConnection(url.OrgURL, cfg.gitPat)
	client, err := git.NewClient(ctx, connection)
	g.Expect(err).ToNot(HaveOccurred())
	getArgs := git.GetStatusesArgs{
		Project:      &url.Project,
		RepositoryId: &url.Repo,
		CommitId:     &rev,
	}
	g.Eventually(func() bool {
		statuses, err := client.GetStatuses(ctx, getArgs)
		if err != nil {
			return false
		}
		if len(*statuses) != 1 {
			return false
		}
		return true
	}, 500*time.Second, 5*time.Second)
}

type AzureDevOpsURL struct {
	OrgURL  string
	Project string
	Repo    string
}

// TODO(somtochiama): move this into fluxcd/pkg and reuse in NC
func ParseAzureDevopsURL(s string) (AzureDevOpsURL, error) {
	var args AzureDevOpsURL
	u, err := giturls.Parse(s)
	if err != nil {
		return args, nil
	}

	scheme := u.Scheme
	if u.Scheme == "ssh" {
		scheme = "https"
	}

	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")

	comp := strings.Split(id, "/")
	if len(comp) != 4 {
		return args, fmt.Errorf("invalid repository id %q", id)
	}

	args = AzureDevOpsURL{
		OrgURL:  fmt.Sprintf("%s://%s/%s", scheme, u.Host, comp[0]),
		Project: comp[1],
		Repo:    comp[3],
	}

	return args, nil

}
