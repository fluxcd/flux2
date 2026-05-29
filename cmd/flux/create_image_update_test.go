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

import "testing"

func TestCreateImageUpdate(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "no signing key",
			args:   "create image update flux-system --git-repo-ref=flux-system --checkout-branch=main --author-name=flux --author-email=flux@example.com --interval=1m0s --namespace=flux-system --export",
			assert: assertGoldenFile("./testdata/create_image_update/no-signing.yaml"),
		},
		{
			name:   "signing secret without explicit type defaults to gpg",
			args:   "create image update flux-system --git-repo-ref=flux-system --checkout-branch=main --author-name=flux --author-email=flux@example.com --signing-key-secret=my-key --interval=1m0s --namespace=flux-system --export",
			assert: assertGoldenFile("./testdata/create_image_update/signing-default-gpg.yaml"),
		},
		{
			name:   "ssh signing key",
			args:   "create image update flux-system --git-repo-ref=flux-system --checkout-branch=main --author-name=flux --author-email=flux@example.com --signing-key-secret=my-deploy-key --signing-key-type=ssh --interval=1m0s --namespace=flux-system --export",
			assert: assertGoldenFile("./testdata/create_image_update/signing-ssh.yaml"),
		},
		{
			name:   "signing-key-type without secret errors",
			args:   "create image update flux-system --git-repo-ref=flux-system --checkout-branch=main --author-name=flux --author-email=flux@example.com --signing-key-type=ssh --namespace=flux-system --export",
			assert: assertError("--signing-key-type requires --signing-key-secret"),
		},
		{
			name:   "invalid signing-key-type errors",
			args:   "create image update flux-system --git-repo-ref=flux-system --checkout-branch=main --author-name=flux --author-email=flux@example.com --signing-key-secret=k --signing-key-type=pgp --namespace=flux-system --export",
			assert: assertError("--signing-key-type must be one of: gpg, ssh"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args,
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
