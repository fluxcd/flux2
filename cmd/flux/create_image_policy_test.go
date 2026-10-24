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

func TestCreateImagePolicy(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "semver and alpha conflict",
			args:   "create image policy podinfo --image-ref=podinfo --select-semver='>=1.0.0' --select-alpha=desc --namespace=flux-system --export",
			assert: assertError("only one of --select-semver, --select-alpha or --select-numeric can be specified"),
		},
		{
			name:   "semver and numeric conflict",
			args:   "create image policy podinfo --image-ref=podinfo --select-semver='>=1.0.0' --select-numeric=desc --namespace=flux-system --export",
			assert: assertError("only one of --select-semver, --select-alpha or --select-numeric can be specified"),
		},
		{
			name:   "alpha and numeric conflict",
			args:   "create image policy podinfo --image-ref=podinfo --select-alpha=desc --select-numeric=desc --namespace=flux-system --export",
			assert: assertError("only one of --select-semver, --select-alpha or --select-numeric can be specified"),
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
