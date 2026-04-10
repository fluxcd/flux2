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
	"testing"
)

func TestCreateReceiverSecret(t *testing.T) {
	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "missing type",
			args:   "create secret receiver test-secret --token=t --hostname=h",
			assert: assertError("--type is required"),
		},
		{
			name:   "invalid type",
			args:   "create secret receiver test-secret --type=invalid --token=t --hostname=h",
			assert: assertError("invalid argument \"invalid\" for \"--type\" flag: receiver type 'invalid' is not supported, must be one of: generic, generic-hmac, github, gitlab, bitbucket, harbor, dockerhub, quay, gcr, nexus, acr, cdevents"),
		},
		{
			name:   "missing hostname",
			args:   "create secret receiver test-secret --type=github --token=t",
			assert: assertError("--hostname is required"),
		},
		{
			name:   "gcr missing email-claim",
			args:   "create secret receiver test-secret --type=gcr --token=t --hostname=h",
			assert: assertError("--email-claim is required for gcr receiver type"),
		},
		{
			name:   "github receiver secret",
			args:   "create secret receiver receiver-secret --type=github --token=test-token --hostname=flux.example.com --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/receiver/secret-receiver.yaml"),
		},
		{
			name:   "gcr receiver secret",
			args:   "create secret receiver gcr-secret --type=gcr --token=test-token --hostname=flux.example.com --email-claim=sa@project.iam.gserviceaccount.com --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/receiver/secret-receiver-gcr.yaml"),
		},
		{
			name:   "gcr receiver secret with custom audience",
			args:   "create secret receiver gcr-secret --type=gcr --token=test-token --hostname=flux.example.com --email-claim=sa@project.iam.gserviceaccount.com --audience-claim=https://custom.audience.example.com --namespace=my-namespace --export",
			assert: assertGoldenFile("testdata/create_secret/receiver/secret-receiver-gcr-audience.yaml"),
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
