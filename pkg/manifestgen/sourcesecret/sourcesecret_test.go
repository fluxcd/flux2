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

// Test_buildGitSecret_BasicAuthFields covers the regression reported in
// https://github.com/fluxcd/flux2/issues/3892: providing just --password
// (e.g. an Azure DevOps PAT, which ignores the username) must still
// produce a secret containing the password field.
func Test_buildGitSecret_BasicAuthFields(t *testing.T) {
	tests := []struct {
		name         string
		opts         Options
		wantUsername string
		wantPassword string
		wantHasUser  bool
		wantHasPass  bool
	}{
		{
			name:         "username and password",
			opts:         Options{Username: "git", Password: "pw"},
			wantUsername: "git",
			wantPassword: "pw",
			wantHasUser:  true,
			wantHasPass:  true,
		},
		{
			name:         "password only (Azure DevOps PAT)",
			opts:         Options{Password: "pat-token"},
			wantPassword: "pat-token",
			wantHasUser:  false,
			wantHasPass:  true,
		},
		{
			name:         "username only",
			opts:         Options{Username: "git"},
			wantUsername: "git",
			wantHasUser:  true,
			wantHasPass:  false,
		},
		{
			name:        "no credentials",
			opts:        Options{},
			wantHasUser: false,
			wantHasPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := buildGitSecret(nil, nil, tt.opts)
			gotUser, hasUser := secret.StringData[UsernameSecretKey]
			gotPass, hasPass := secret.StringData[PasswordSecretKey]
			if hasUser != tt.wantHasUser {
				t.Errorf("username presence = %v, want %v", hasUser, tt.wantHasUser)
			}
			if hasPass != tt.wantHasPass {
				t.Errorf("password presence = %v, want %v", hasPass, tt.wantHasPass)
			}
			if tt.wantHasUser && gotUser != tt.wantUsername {
				t.Errorf("username = %q, want %q", gotUser, tt.wantUsername)
			}
			if tt.wantHasPass && gotPass != tt.wantPassword {
				t.Errorf("password = %q, want %q", gotPass, tt.wantPassword)
			}
		})
	}
}
