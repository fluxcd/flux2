//go:build e2e
// +build e2e

/*
Copyright 2025 The Flux authors

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

func TestCreateTenant(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "no args",
			args:   "create tenant",
			assert: assertError("name is required"),
		},
		{
			name:   "no namespace",
			args:   "create tenant dev-team --cluster-role=cluster-admin",
			assert: assertError("with-namespace is required"),
		},
		{
			name:   "basic tenant",
			args:   "create tenant dev-team --with-namespace=apps --cluster-role=cluster-admin --export",
			assert: assertGoldenFile("./testdata/create_tenant/tenant-basic.yaml"),
		},
		{
			name:   "tenant with custom serviceaccount",
			args:   "create tenant dev-team --with-namespace=apps --cluster-role=cluster-admin --with-service-account=flux-tenant --export",
			assert: assertGoldenFile("./testdata/create_tenant/tenant-with-service-account.yaml"),
		},
		{
			name:   "tenant with custom cluster role",
			args:   "create tenant dev-team --with-namespace=apps --cluster-role=custom-role --export",
			assert: assertGoldenFile("./testdata/create_tenant/tenant-with-cluster-role.yaml"),
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
