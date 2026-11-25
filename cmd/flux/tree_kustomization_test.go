//go:build unit
// +build unit

/*
Copyright 2021 The Flux authors

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

package main

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"

	"github.com/fluxcd/cli-utils/pkg/object"
	"github.com/fluxcd/flux2/v2/internal/tree"
)

func TestTree(t *testing.T) {
	cases := []struct {
		name       string
		args       string
		objectFile string
		goldenFile string
	}{
		{
			"tree kustomization",
			"tree kustomization flux-system",
			"testdata/tree/kustomizations.yaml",
			"testdata/tree/tree.golden",
		},
		{
			"tree kustomization compact",
			"tree kustomization flux-system --compact",
			"testdata/tree/kustomizations.yaml",
			"testdata/tree/tree-compact.golden",
		},
		{
			"tree kustomization empty",
			"tree kustomization empty",
			"testdata/tree/kustomizations.yaml",
			"testdata/tree/tree-empty.golden",
		},
		{
			"tree kustomization escapes control characters",
			"tree kustomization escape",
			"testdata/tree/kustomizations.yaml",
			"testdata/tree/tree-escape.golden",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := map[string]string{
				"fluxns": allocateNamespace("flux-system"),
			}
			testEnv.CreateObjectFile(tc.objectFile, tmpl, t)
			cmd := cmdTestCase{
				args:   tc.args + " -n=" + tmpl["fluxns"],
				assert: assertGoldenTemplateFile(tc.goldenFile, tmpl),
			}
			cmd.runTestCmd(t)
		})
	}
}

func TestTreeKustomizationFullFlags(t *testing.T) {
	outputFlag := treeKsCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("expected the output flag to be registered")
	}

	managedFieldsFlag := treeKsCmd.Flags().Lookup("show-managed-fields")
	if managedFieldsFlag == nil {
		t.Fatal("expected the show-managed-fields flag to be registered")
	}
	if managedFieldsFlag.DefValue != "false" {
		t.Fatalf("expected show-managed-fields to default to false, got %q", managedFieldsFlag.DefValue)
	}
}

func TestKustomizationToUnstructured(t *testing.T) {
	k := &kustomizev1.Kustomization{}
	k.SetName("podinfo")
	k.SetNamespace("flux-system")

	u, err := kustomizationToUnstructured(k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.GetAPIVersion() != kustomizev1.GroupVersion.String() {
		t.Fatalf("unexpected apiVersion: %q", u.GetAPIVersion())
	}
	if u.GetKind() != kustomizev1.KustomizationKind {
		t.Fatalf("unexpected kind: %q", u.GetKind())
	}
	if u.GetName() != "podinfo" || u.GetNamespace() != "flux-system" {
		t.Fatalf("unexpected object key: %s/%s", u.GetNamespace(), u.GetName())
	}
}

func TestStripManagedFields(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name": "podinfo",
			"managedFields": []any{
				map[string]any{"manager": "kustomize-controller"},
			},
		},
	}}

	stripManagedFields(u)

	metadata, found, err := unstructured.NestedMap(u.Object, "metadata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected metadata")
	}
	if _, found := metadata["managedFields"]; found {
		t.Fatal("expected managedFields to be removed")
	}
}

func TestGetFullObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core types to scheme: %v", err)
	}

	configMap := &corev1.ConfigMap{}
	configMap.SetName("podinfo")
	configMap.SetNamespace("apps")
	configMap.Data = map[string]string{"key": "value"}

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()
	gvk := schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
	meta := object.ObjMetadata{Name: "podinfo", Namespace: "apps"}

	obj, err := getFullObject(t.Context(), kubeClient, gvk, meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.GetName() != "podinfo" || obj.GetNamespace() != "apps" {
		t.Fatalf("unexpected object key: %s/%s", obj.GetNamespace(), obj.GetName())
	}
	if value, found, err := unstructured.NestedString(obj.Object, "data", "key"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !found || value != "value" {
		t.Fatalf("expected ConfigMap data, got value=%q found=%v", value, found)
	}

	missingMeta := object.ObjMetadata{Name: "missing", Namespace: "apps"}
	fallback, err := getFullObject(t.Context(), kubeClient, gvk, missingMeta)
	if err != nil {
		t.Fatalf("unexpected fallback error: %v", err)
	}
	if fallback.GetAPIVersion() != "v1" || fallback.GetKind() != "ConfigMap" {
		t.Fatalf("unexpected fallback GVK: %s %s", fallback.GetAPIVersion(), fallback.GetKind())
	}
	if fallback.GetName() != "missing" {
		t.Fatalf("unexpected fallback name: %q", fallback.GetName())
	}
}

func TestFullResourceTreeYAML(t *testing.T) {
	resource := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      "podinfo",
			"namespace": "apps",
		},
	}}
	kTree := tree.NewResource(resource)
	kTree.AddResource(&unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name":      "podinfo",
			"namespace": "apps",
		},
	}})

	data, err := yaml.Marshal(kTree)
	if err != nil {
		t.Fatalf("failed to marshal full resource tree: %v", err)
	}
	output := string(data)
	for _, expected := range []string{
		"resource:",
		"apiVersion: v1",
		"kind: ConfigMap",
		"resources:",
		"apiVersion: apps/v1",
		"kind: Deployment",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected full resource tree output to contain %q:\n%s", expected, output)
		}
	}
}
