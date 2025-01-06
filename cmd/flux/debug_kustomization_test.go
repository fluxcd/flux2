//go:build unit
// +build unit

/*
Copyright 2024 The Flux authors

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

func TestDebugKustomization(t *testing.T) {
	namespace := allocateNamespace("debug")

	objectFile := "testdata/debug_kustomization/objects.yaml"
	tmpl := map[string]string{
		"fluxns": namespace,
	}
	testEnv.CreateObjectFile(objectFile, tmpl, t)

	cases := []struct {
		name       string
		arg        string
		goldenFile string
		tmpl       map[string]string
	}{
		{
			"debug status",
			"debug ks test --show-status --show-vars=false",
			"testdata/debug_kustomization/status.golden.yaml",
			tmpl,
		},
		{
			"debug vars",
			"debug ks test --show-vars --show-status=false",
			"testdata/debug_kustomization/vars.golden.env",
			tmpl,
		},
		{
			"debug vars from",
			"debug ks test-from --show-vars --show-status=false",
			"testdata/debug_kustomization/vars-from.golden.env",
			tmpl,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.arg + " -n=" + namespace,
				assert: assertGoldenTemplateFile(tt.goldenFile, tmpl),
			}

			cmd.runTestCmd(t)
		})
	}
}
