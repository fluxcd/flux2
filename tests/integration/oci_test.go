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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

func TestOCIHelmRelease(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()

	// Create namespace for test
	testID := "oci-helm-" + randStringRunes(5)
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testID,
		},
	}
	g.Expect(testEnv.Create(ctx, &namespace)).To(Succeed())
	defer testEnv.Delete(ctx, &namespace)

	repoURL := fmt.Sprintf("%s/charts/podinfo", cfg.testRegistry)
	err := pushImagesFromURL(repoURL, "ghcr.io/stefanprodan/charts/podinfo:6.2.0", []string{"6.2.0"})
	g.Expect(err).ToNot(HaveOccurred())

	// Create HelmRepository.
	helmRepository := sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{Name: testID, Namespace: testID},
		Spec: sourcev1.HelmRepositorySpec{
			URL: fmt.Sprintf("oci://%s", cfg.testRegistry),
			Interval: metav1.Duration{
				Duration: 5 * time.Minute,
			},
			Provider:        infraOpts.Provider,
			PassCredentials: true,
			Type:            "oci",
		},
	}

	g.Expect(testEnv.Create(ctx, &helmRepository)).To(Succeed())
	defer testEnv.Delete(ctx, &helmRepository)

	// create helm release
	helmRelease := helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: testID, Namespace: testID},
		Spec: helmv2.HelmReleaseSpec{
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Interval: &metav1.Duration{
						Duration: 10 * time.Minute,
					},
					Chart:   "charts/podinfo",
					Version: "6.2.0",
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind:      sourcev1.HelmRepositoryKind,
						Name:      helmRepository.Name,
						Namespace: helmRepository.Namespace,
					},
				},
			},
		},
	}

	g.Expect(testEnv.Create(ctx, &helmRelease)).To(Succeed())
	defer testEnv.Delete(ctx, &helmRelease)

	g.Eventually(func() bool {
		chart := &sourcev1.HelmChart{}
		nn := types.NamespacedName{
			Name:      fmt.Sprintf("%s-%s", helmRelease.Name, helmRelease.Namespace),
			Namespace: helmRelease.Namespace,
		}
		if err := testEnv.Get(ctx, nn, chart); err != nil {
			t.Logf("error getting helm chart %s\n", err.Error())
			return false
		}
		if err := checkReadyCondition(chart); err != nil {
			t.Log(err)
			return false
		}

		obj := &helmv2.HelmRelease{}
		nn = types.NamespacedName{Name: helmRelease.Name, Namespace: helmRelease.Namespace}
		if err := testEnv.Get(ctx, nn, obj); err != nil {
			t.Logf("error getting helm release %s\n", err.Error())
			return false
		}

		if err := checkReadyCondition(obj); err != nil {
			t.Log(err)
			return false
		}

		return true
	}, testTimeout, testInterval).Should(BeTrue())
}
