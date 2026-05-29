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
	"strings"
	"testing"
)

func TestBootstrapValidate_signingFlags(t *testing.T) {
	tests := []struct {
		name       string
		gpgRing    string
		gpgPass    string
		sshKey     string
		sshPass    string
		sshPassp   string
		privateKey string
		reuse      bool
		wantErr    string
	}{
		{name: "no signing flags is valid"},
		{name: "GPG only is valid", gpgRing: "./testdata/bootstrap/gpg.pgp"},
		{name: "SSH only is valid", sshKey: "./testdata/bootstrap/ed25519.private"},
		{
			name:       "Reuse-private-key with private-key-file is valid",
			privateKey: "./testdata/bootstrap/ed25519.private",
			reuse:      true,
		},
		{
			name:    "GPG + SSH errors",
			gpgRing: "./testdata/bootstrap/gpg.pgp",
			sshKey:  "./testdata/bootstrap/ed25519.private",
			wantErr: "--gpg-* and --ssh-signing-* are mutually exclusive",
		},
		{
			name:       "GPG + reuse errors",
			gpgRing:    "./testdata/bootstrap/gpg.pgp",
			privateKey: "./testdata/bootstrap/ed25519.private",
			reuse:      true,
			wantErr:    "--gpg-* and --ssh-signing-* are mutually exclusive",
		},
		{
			name:       "SSH key-file + reuse errors",
			sshKey:     "./testdata/bootstrap/ed25519.private",
			privateKey: "./testdata/bootstrap/ed25519.private",
			reuse:      true,
			wantErr:    "--ssh-signing-key-file and --ssh-signing-reuse-private-key are mutually exclusive",
		},
		{
			name:    "Reuse without private-key-file errors",
			reuse:   true,
			wantErr: "--ssh-signing-reuse-private-key requires --private-key-file",
		},
		{
			name:    "SSH password without key errors",
			sshPass: "secret",
			wantErr: "--ssh-signing-password requires --ssh-signing-key-file",
		},
		{
			name:     "SSH passphrase alias alone applies",
			sshKey:   "./testdata/bootstrap/ed25519-encrypted.private",
			sshPassp: "abcde12345",
		},
		{
			name:     "SSH password and passphrase with same value passes",
			sshKey:   "./testdata/bootstrap/ed25519-encrypted.private",
			sshPass:  "abcde12345",
			sshPassp: "abcde12345",
		},
		{
			name:     "SSH password and passphrase with different values errors",
			sshKey:   "./testdata/bootstrap/ed25519-encrypted.private",
			sshPass:  "right",
			sshPassp: "wrong",
			wantErr:  "are aliases; do not pass both",
		},
		{
			name:    "SSH malformed key fails pre-flight",
			sshKey:  "./testdata/bootstrap/malformed.private",
			wantErr: "invalid SSH signing key",
		},
		{
			name:    "SSH encrypted key without password fails pre-flight",
			sshKey:  "./testdata/bootstrap/ed25519-encrypted.private",
			wantErr: "passphrase required",
		},
		// The GPG fixture used here is encrypted (passphrase: "right") so that
		// passing the wrong passphrase exercises the Decrypt error path.
		// An unencrypted key would make Decrypt a no-op regardless of the
		// passphrase supplied.
		{
			name:    "GPG with wrong passphrase fails pre-flight",
			gpgRing: "./testdata/bootstrap/gpg-encrypted.pgp",
			gpgPass: "wrong",
			wantErr: "invalid GPG signing key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			savedGpgRing := bootstrapArgs.gpgKeyRingPath
			savedGpgPass := bootstrapArgs.gpgPassphrase
			savedSshKey := bootstrapArgs.sshSigningKeyFile
			savedSshPass := bootstrapArgs.sshSigningPassword
			savedSshPassp := bootstrapArgs.sshSigningPassphrase
			savedPrivKey := bootstrapArgs.privateKeyFile
			savedReuse := bootstrapArgs.sshSigningReusePrivateKey
			defer func() {
				bootstrapArgs.gpgKeyRingPath = savedGpgRing
				bootstrapArgs.gpgPassphrase = savedGpgPass
				bootstrapArgs.sshSigningKeyFile = savedSshKey
				bootstrapArgs.sshSigningPassword = savedSshPass
				bootstrapArgs.sshSigningPassphrase = savedSshPassp
				bootstrapArgs.privateKeyFile = savedPrivKey
				bootstrapArgs.sshSigningReusePrivateKey = savedReuse
			}()

			bootstrapArgs.gpgKeyRingPath = tt.gpgRing
			bootstrapArgs.gpgPassphrase = tt.gpgPass
			bootstrapArgs.sshSigningKeyFile = tt.sshKey
			bootstrapArgs.sshSigningPassword = tt.sshPass
			bootstrapArgs.sshSigningPassphrase = tt.sshPassp
			bootstrapArgs.privateKeyFile = tt.privateKey
			bootstrapArgs.sshSigningReusePrivateKey = tt.reuse

			err := bootstrapValidate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
