//go:build unit
// +build unit

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

package main

import (
	"context"
	"slices"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

func TestDeleteMultipleNames(t *testing.T) {
	cases := []struct {
		name       string
		args       string
		goldenFile string
		wantErr    string
		remaining  []string
	}{
		{
			name:       "single name",
			args:       "delete kustomization app-a",
			goldenFile: "testdata/delete/single.golden",
			remaining:  []string{"app-b", "app-c"},
		},
		{
			name:    "single missing name",
			args:    "delete kustomization missing",
			wantErr: `kustomizations.kustomize.toolkit.fluxcd.io "missing" not found`,
			remaining: []string{
				"app-a", "app-b", "app-c",
			},
		},
		{
			name:       "two names",
			args:       "delete kustomization app-a app-b",
			goldenFile: "testdata/delete/two.golden",
			remaining:  []string{"app-c"},
		},
		{
			name:       "order and duplicates",
			args:       "delete kustomization app-b app-a app-b",
			goldenFile: "testdata/delete/duplicates.golden",
			remaining:  []string{"app-c"},
		},
		{
			name:       "one missing among several",
			args:       "delete kustomization app-a missing app-c",
			goldenFile: "testdata/delete/missing.golden",
			wantErr:    "failed to delete 1 of 3 kustomization objects",
			remaining:  []string{"app-b"},
		},
		{
			name:       "all missing",
			args:       "delete kustomization missing-a missing-b",
			goldenFile: "testdata/delete/all-missing.golden",
			wantErr:    "failed to delete 2 of 2 kustomization objects",
			remaining:  []string{"app-a", "app-b", "app-c"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			namespace := allocateNamespace("delete-multi")
			tmpl := map[string]string{
				"fluxns": namespace,
			}
			testEnv.CreateObjectFile("testdata/delete/objects.yaml", tmpl, t)

			cmd := cmdTestCase{
				args:   tt.args + " --silent -n=" + namespace,
				assert: assertGoldenTemplateFileAndError(tt.goldenFile, tmpl, tt.wantErr),
			}
			cmd.runTestCmd(t)

			var list kustomizev1.KustomizationList
			if err := testEnv.client.List(context.Background(), &list, client.InNamespace(namespace)); err != nil {
				t.Fatalf("failed to list kustomizations: %v", err)
			}
			var names []string
			for _, ks := range list.Items {
				names = append(names, ks.Name)
			}
			slices.Sort(names)
			if !slices.Equal(names, tt.remaining) {
				t.Errorf("expected remaining kustomizations %v, got %v", tt.remaining, names)
			}
		})
	}
}
