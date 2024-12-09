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

func TestCreateSecretGitHubApp(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "create githubapp secret with missing name",
			args:   "create secret githubapp",
			assert: assertError("name is required"),
		},
		{
			name:   "create githubapp secret with missing app-id",
			args:   "create secret githubapp appinfo",
			assert: assertError("--app-id is required"),
		},
		{
			name:   "create githubapp secret with missing appInstallationID",
			args:   "create secret githubapp appinfo --app-id 1",
			assert: assertError("--app-installation-id is required"),
		},
		{
			name:   "create githubapp secret with missing private key file",
			args:   "create secret githubapp appinfo --app-id 1 --app-installation-id 2",
			assert: assertError("--app-private-key is required"),
		},
		{
			name:   "create githubapp secret with private key file that does not exist",
			args:   "create secret githubapp appinfo --app-id 1 --app-installation-id 2 --app-private-key pk.pem",
			assert: assertError("unable to read private key file: open pk.pem: no such file or directory"),
		},
		{
			name:   "create githubapp secret with app info",
			args:   "create secret githubapp appinfo --namespace my-namespace --app-id 1 --app-installation-id 2 --app-private-key ./testdata/create_secret/githubapp/test-private-key.pem --export",
			assert: assertGoldenFile("testdata/create_secret/githubapp/secret.yaml"),
		},
		{
			name:   "create githubapp secret with appinfo and base url",
			args:   "create secret githubapp appinfo --namespace my-namespace --app-id 1 --app-installation-id 2 --app-private-key ./testdata/create_secret/githubapp/test-private-key.pem --app-base-url www.example.com/api/v3 --export",
			assert: assertGoldenFile("testdata/create_secret/githubapp/secret-with-baseurl.yaml"),
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
