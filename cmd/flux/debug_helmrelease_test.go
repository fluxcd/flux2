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

func TestDebugHelmRelease(t *testing.T) {
	namespace := allocateNamespace("debug")

	objectFile := "testdata/debug_helmrelease/objects.yaml"
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
			"debug helmrelease test-values-inline --show-status --show-values=false",
			"testdata/debug_helmrelease/status.golden.yaml",
			tmpl,
		},
		{
			"debug values",
			"debug helmrelease test-values-inline --show-values --show-status=false",
			"testdata/debug_helmrelease/values-inline.golden.yaml",
			tmpl,
		},
		{
			"debug values from",
			"debug helmrelease test-values-from --show-values --show-status=false",
			"testdata/debug_helmrelease/values-from.golden.yaml",
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
