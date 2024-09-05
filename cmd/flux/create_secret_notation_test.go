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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	trustPolicy        = "./testdata/create_secret/notation/test-trust-policy.json"
	invalidTrustPolicy = "./testdata/create_secret/notation/invalid-trust-policy.json"
	invalidJson        = "./testdata/create_secret/notation/invalid.json"
	testCertFolder     = "./testdata/create_secret/notation"
)

func TestCreateNotationSecret(t *testing.T) {
	crt, err := os.Create(filepath.Join(t.TempDir(), "ca.crt"))
	if err != nil {
		t.Fatal("could not create ca.crt file")
	}

	pem, err := os.Create(filepath.Join(t.TempDir(), "ca.pem"))
	if err != nil {
		t.Fatal("could not create ca.pem file")
	}

	invalidCert, err := os.Create(filepath.Join(t.TempDir(), "ca.p12"))
	if err != nil {
		t.Fatal("could not create ca.p12 file")
	}

	_, err = crt.Write([]byte("ca-data-crt"))
	if err != nil {
		t.Fatal("could not write to crt certificate file")
	}

	_, err = pem.Write([]byte("ca-data-pem"))
	if err != nil {
		t.Fatal("could not write to pem certificate file")
	}

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "no args",
			args:   "create secret notation",
			assert: assertError("name is required"),
		},
		{
			name:   "no trust policy",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s", testCertFolder),
			assert: assertError("--trust-policy-file is required"),
		},
		{
			name:   "no cert",
			args:   fmt.Sprintf("create secret notation notation-config --trust-policy-file=%s", trustPolicy),
			assert: assertError("--ca-cert-file is required"),
		},
		{
			name:   "non pem and crt cert",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --trust-policy-file=%s", invalidCert.Name(), trustPolicy),
			assert: assertError("ca.p12 must end with either .crt or .pem"),
		},
		{
			name:   "invalid trust policy",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --trust-policy-file=%s", t.TempDir(), invalidTrustPolicy),
			assert: assertError("invalid trust policy: trust policy: a trust policy statement is missing a name, every statement requires a name"),
		},
		{
			name:   "invalid trust policy json",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --trust-policy-file=%s", t.TempDir(), invalidJson),
			assert: assertError(fmt.Sprintf("failed to unmarshal trust policy %s: json: cannot unmarshal string into Go value of type trustpolicy.Document", invalidJson)),
		},
		{
			name:   "crt secret",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --trust-policy-file=%s --namespace=my-namespace --export", crt.Name(), trustPolicy),
			assert: assertGoldenFile("./testdata/create_secret/notation/secret-ca-crt.yaml"),
		},
		{
			name:   "pem secret",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --trust-policy-file=%s --namespace=my-namespace --export", pem.Name(), trustPolicy),
			assert: assertGoldenFile("./testdata/create_secret/notation/secret-ca-pem.yaml"),
		},
		{
			name:   "multi secret",
			args:   fmt.Sprintf("create secret notation notation-config --ca-cert-file=%s --ca-cert-file=%s --trust-policy-file=%s --namespace=my-namespace --export", crt.Name(), pem.Name(), trustPolicy),
			assert: assertGoldenFile("./testdata/create_secret/notation/secret-ca-multi.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				secretNotationArgs = secretNotationFlags{}
			}()

			cmd := cmdTestCase{
				args:   tt.args,
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
