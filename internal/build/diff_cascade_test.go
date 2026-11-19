//go:build unit

/*
Copyright 2026 The Flux authors

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

package build

import (
	"context"
	"strings"
	"testing"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeletedKustomizationDiff(t *testing.T) {
	tests := []struct {
		name        string
		prune       bool
		policy      string
		suspend     bool
		owner       string
		annotations map[string]string
		want        string
	}{
		{name: "default prune", prune: true, owner: "parent", want: "deleted"},
		{name: "default orphan", owner: "parent"},
		{name: "mirror prune", prune: true, policy: kustomizev1.DeletionPolicyMirrorPrune, owner: "parent", want: "deleted"},
		{name: "mirror orphan", policy: kustomizev1.DeletionPolicyMirrorPrune, owner: "parent"},
		{name: "explicit orphan", prune: true, policy: kustomizev1.DeletionPolicyOrphan, owner: "parent"},
		{name: "explicit delete", policy: kustomizev1.DeletionPolicyDelete, owner: "parent", want: "deleted"},
		{name: "wait for termination", policy: kustomizev1.DeletionPolicyWaitForTermination, owner: "parent", want: "deleted"},
		{name: "suspended", prune: true, suspend: true, owner: "parent"},
		{name: "ownership changed", prune: true, owner: "other", want: "skipped"},
		{name: "pruning disabled", prune: true, owner: "parent", annotations: map[string]string{controllerGroup + "/prune": kustomizev1.DisabledValue}, want: "skipped"},
		{name: "reconciliation disabled", prune: true, owner: "parent", annotations: map[string]string{controllerGroup + "/reconcile": kustomizev1.DisabledValue}, want: "skipped"},
		{name: "ignored", prune: true, owner: "parent", annotations: map[string]string{controllerGroup + "/ssa": kustomizev1.IgnoreValue}, want: "skipped"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{Name: "parent", Namespace: "test"},
				Spec:       kustomizev1.KustomizationSpec{Prune: tt.prune, DeletionPolicy: tt.policy, Suspend: tt.suspend},
				Status:     kustomizev1.KustomizationStatus{Inventory: &kustomizev1.ResourceInventory{Entries: []kustomizev1.ResourceRef{{ID: "test_config__ConfigMap", Version: "v1"}}}},
			}
			config := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: "config", Namespace: "test", Annotations: tt.annotations,
				Labels: map[string]string{controllerGroup + "/name": tt.owner, controllerGroup + "/namespace": "test"},
			}}
			scheme := runtime.NewScheme()
			if err := kustomizev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			b := &Builder{client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(ks, config).Build()}
			obj := &unstructured.Unstructured{}
			obj.SetAPIVersion(kustomizev1.GroupVersion.String())
			obj.SetKind("Kustomization")
			obj.SetName(ks.Name)
			obj.SetNamespace(ks.Namespace)
			got, err := b.deletedKustomizationDiff(context.Background(), obj, make(map[types.NamespacedName]struct{}))
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if got != "" {
					t.Fatalf("unexpected cascade output: %s", got)
				}
			} else if !strings.Contains(got, "ConfigMap/test/config "+tt.want) {
				t.Fatalf("expected resource to be %s, got: %s", tt.want, got)
			}
		})
	}
}

func TestDeletedKustomizationDiffCycle(t *testing.T) {
	ks := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{Name: "cycle", Namespace: "test", Labels: map[string]string{controllerGroup + "/name": "cycle", controllerGroup + "/namespace": "test"}},
		Spec:       kustomizev1.KustomizationSpec{Prune: true},
		Status:     kustomizev1.KustomizationStatus{Inventory: &kustomizev1.ResourceInventory{Entries: []kustomizev1.ResourceRef{{ID: "test_cycle_kustomize.toolkit.fluxcd.io_Kustomization", Version: "v1"}}}},
	}
	scheme := runtime.NewScheme()
	if err := kustomizev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	b := &Builder{client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(ks).Build()}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(kustomizev1.GroupVersion.String())
	obj.SetKind("Kustomization")
	obj.SetName(ks.Name)
	obj.SetNamespace(ks.Namespace)
	got, err := b.deletedKustomizationDiff(context.Background(), obj, make(map[types.NamespacedName]struct{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got, "deleted resources") != 1 {
		t.Fatalf("cycle was traversed more than once: %s", got)
	}
}
