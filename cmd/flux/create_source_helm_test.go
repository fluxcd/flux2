//go:build unit
// +build unit

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

package main

import (
	"testing"
)

func TestCreateSourceHelm(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		resultFile string
		assertFunc string
	}{
		{
			name:       "no args",
			args:       "create source helm",
			resultFile: "name is required",
			assertFunc: "assertError",
		},
		{
			name:       "OCI repo",
			args:       "create source helm podinfo --url=oci://ghcr.io/stefanprodan/charts/podinfo --interval 5m --export",
			resultFile: "./testdata/create_source_helm/oci.golden",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "OCI repo with Secret ref",
			args:       "create source helm podinfo --url=oci://ghcr.io/stefanprodan/charts/podinfo --interval 5m --secret-ref=creds --export",
			resultFile: "./testdata/create_source_helm/oci-with-secret.golden",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "HTTPS repo",
			args:       "create source helm podinfo --url=https://stefanprodan.github.io/charts/podinfo --interval 5m --export",
			resultFile: "./testdata/create_source_helm/https.golden",
			assertFunc: "assertGoldenTemplateFile",
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setup(t, tmpl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var assert assertFunc
			switch tt.assertFunc {
			case "assertGoldenTemplateFile":
				assert = assertGoldenTemplateFile(tt.resultFile, tmpl)
			case "assertError":
				assert = assertError(tt.resultFile)
			}

			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
