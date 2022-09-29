/*
Copyright 2022 The Flux authors

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

package test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

func TestACRHelmRelease(t *testing.T) {
	g := NewWithT(t)
	ctx := context.TODO()

	// Create namespace for test
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acr-helm-release",
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, testEnv.Client, &namespace, func() error {
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	repoURL := fmt.Sprintf("%s/charts/podinfo", cfg.dockerCred.url)
	err = pushImagesFromURL(repoURL, "ghcr.io/stefanprodan/charts/podinfo:6.2.0", []string{"v0.0.1"})
	g.Expect(err).ToNot(HaveOccurred())

	// Create HelmRepository and wait for it to sync
	helmRepository := sourcev1.HelmRepository{ObjectMeta: metav1.ObjectMeta{Name: "acr", Namespace: namespace.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, cfg.client, &helmRepository, func() error {
		helmRepository.Spec = sourcev1.HelmRepositorySpec{
			URL: fmt.Sprintf("oci://%s", repoURL),
			Interval: metav1.Duration{
				Duration: 5 * time.Minute,
			},
			Provider:        "azure",
			PassCredentials: true,
			Type:            "oci",
		}
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Eventually(func() bool {
		obj := &sourcev1.HelmRepository{}
		nn := types.NamespacedName{Name: helmRepository.Name, Namespace: helmRepository.Namespace}
		err := testEnv.Client.Get(ctx, nn, obj)
		if err != nil {
			log.Printf("error getting helm repository %s\n", err.Error())
			return false
		}
		if apimeta.IsStatusConditionPresentAndEqual(obj.Status.Conditions, meta.ReadyCondition, metav1.ConditionTrue) == false {
			log.Println("helm repository condition not ready")
			return false
		}
		return true
	}, 30*time.Second, 5*time.Second)
}
