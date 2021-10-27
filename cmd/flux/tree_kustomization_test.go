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
	"testing"
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
