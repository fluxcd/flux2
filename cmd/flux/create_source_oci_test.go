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

func TestCreateSourceOCI(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		assertFunc assertFunc
	}{
		{
			name:       "NoArgs",
			args:       "create source oci",
			assertFunc: assertError("name is required"),
		},
		{
			name:       "NoURL",
			args:       "create source oci podinfo",
			assertFunc: assertError("url is required"),
		},
		{
			name:       "verify provider not specified",
			args:       "create source oci podinfo --url=oci://ghcr.io/stefanprodan/manifests/podinfo --tag=6.3.5 --verify-secret-ref=cosign-pub",
			assertFunc: assertError("a verification provider must be specified when a secret is specified"),
		},
		{
			name:       "export manifest",
			args:       "create source oci podinfo --url=oci://ghcr.io/stefanprodan/manifests/podinfo --tag=6.3.5 --interval 10m --export",
			assertFunc: assertGoldenFile("./testdata/oci/export.golden"),
		},
		{
			name:       "export manifest with secret",
			args:       "create source oci podinfo --url=oci://ghcr.io/stefanprodan/manifests/podinfo --tag=6.3.5 --interval 10m --secret-ref=creds --export",
			assertFunc: assertGoldenFile("./testdata/oci/export_with_secret.golden"),
		},
		{
			name:       "export manifest with verify secret",
			args:       "create source oci podinfo --url=oci://ghcr.io/stefanprodan/manifests/podinfo --tag=6.3.5 --interval 10m --verify-provider=cosign --verify-secret-ref=cosign-pub --export",
			assertFunc: assertGoldenFile("./testdata/oci/export_with_verify_secret.golden"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args,
				assert: tt.assertFunc,
			}

			cmd.runTestCmd(t)
		})
	}
}
