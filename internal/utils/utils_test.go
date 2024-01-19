//go:build !e2e
// +build !e2e

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

package utils

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fluxcd/pkg/apis/meta"
)

func TestCompatibleVersion(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		target string
		want   bool
	}{
		{"different major version", "1.1.0", "0.1.0", false},
		{"different minor version", "0.1.0", "0.2.0", false},
		{"same version", "0.1.0", "0.1.0", true},
		{"binary patch version ahead", "0.1.1", "0.1.0", true},
		{"target patch version ahead", "0.1.1", "0.1.2", true},
		{"prerelease binary", "0.0.0-dev.0", "0.1.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompatibleVersion(tt.binary, tt.target); got != tt.want {
				t.Errorf("CompatibleVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseObjectKindNameNamespace(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantKind      string
		wantName      string
		wantNamespace string
	}{
		{"with kind name namespace", "Kustomization/foo.flux-system", "Kustomization", "foo", "flux-system"},
		{"without namespace", "Kustomization/foo", "Kustomization", "foo", ""},
		{"name with dots", "Kustomization/foo.bar.flux-system", "Kustomization", "foo.bar", "flux-system"},
		{"multiple slashes", "foo/bar/baz", "", "foo/bar/baz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotName, gotNamespace := ParseObjectKindNameNamespace(tt.input)
			if gotKind != tt.wantKind {
				t.Errorf("kind = %s, want %s", gotKind, tt.wantKind)
			}
			if gotName != tt.wantName {
				t.Errorf("name = %s, want %s", gotName, tt.wantName)
			}
			if gotNamespace != tt.wantNamespace {
				t.Errorf("namespace = %s, want %s", gotNamespace, tt.wantNamespace)
			}
		})
	}
}

func TestMakeDependsOn(t *testing.T) {
	input := []string{
		"someNSA/someNameA",
		"someNSB/someNameB",
		"someNameC",
		"someNSD/",
		"",
	}
	want := []meta.NamespacedObjectReference{
		{Namespace: "someNSA", Name: "someNameA"},
		{Namespace: "someNSB", Name: "someNameB"},
		{Namespace: "", Name: "someNameC"},
		{Namespace: "someNSD", Name: ""},
		{Namespace: "", Name: ""},
	}

	got := MakeDependsOn(input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MakeDependsOn() = %v, want %v", got, want)
	}
}

func TestValidateComponents(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expectErr bool
	}{
		{"default and extra components", []string{"source-controller", "image-reflector-controller"}, false},
		{"unknown components", []string{"some-comp-1", "some-comp-2"}, true},
		{"mix of default and unknown", []string{"source-controller", "some-comp-1"}, true},
		{"empty", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateComponents(tt.input); (err != nil) != tt.expectErr {
				t.Errorf("ValidateComponents() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestExtractCRDs(t *testing.T) {
	tests := []struct {
		name           string
		inManifestFile string
		expectErr      bool
	}{
		{"with crds", "components-with-crds.yaml", false},
		{"without crds", "components-without-crds.yaml", true},
		{"non-existent file", "non-existent-file.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outManifestPath := filepath.Join(t.TempDir(), "crds.yaml")
			inManifestPath := filepath.Join("testdata", tt.inManifestFile)
			if err := ExtractCRDs(inManifestPath, outManifestPath); (err != nil) != tt.expectErr {
				t.Errorf("ExtractCRDs() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
