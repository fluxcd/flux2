//go:build !e2e
// +build !e2e

/*
Copyright 2021 The Flux authors

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

package sourcesecret

import (
	"os"
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/testdata"
)

func Test_passwordLoadKeyPair(t *testing.T) {
	tests := []struct {
		name           string
		privateKeyPath string
		publicKeyPath  string
		password       string
	}{
		{
			name:           "private key pair with password",
			privateKeyPath: "testdata/password_rsa",
			publicKeyPath:  "testdata/password_rsa.pub",
			password:       "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, _ := os.ReadFile(tt.privateKeyPath)
			ppk, _ := os.ReadFile(tt.publicKeyPath)

			got, err := LoadKeyPair(pk, tt.password)
			if err != nil {
				t.Errorf("loadKeyPair() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got.PrivateKey, pk) {
				t.Errorf("PrivateKey %s != %s", got.PrivateKey, pk)
			}
			if !reflect.DeepEqual(got.PublicKey, ppk) {
				t.Errorf("PublicKey %s != %s", got.PublicKey, ppk)
			}
		})
	}
}

func Test_PasswordlessLoadKeyPair(t *testing.T) {
	for algo, privateKey := range testdata.PEMBytes {
		t.Run(algo, func(t *testing.T) {
			got, err := LoadKeyPair(privateKey, "")
			if err != nil {
				t.Errorf("loadKeyPair() error = %v", err)
				return
			}

			if !reflect.DeepEqual(got.PrivateKey, privateKey) {
				t.Errorf("PrivateKey %s != %s", got.PrivateKey, string(privateKey))
			}

			signer, err := ssh.ParsePrivateKey(privateKey)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(got.PublicKey, ssh.MarshalAuthorizedKey(signer.PublicKey())) {
				t.Errorf("PublicKey %s != %s", got.PublicKey, ssh.MarshalAuthorizedKey(signer.PublicKey()))
			}
		})
	}
}
