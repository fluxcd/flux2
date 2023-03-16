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
	"crypto/elliptic"

	"github.com/fluxcd/pkg/ssh"
)

type PrivateKeyAlgorithm string

const (
	RSAPrivateKeyAlgorithm     PrivateKeyAlgorithm = "rsa"
	ECDSAPrivateKeyAlgorithm   PrivateKeyAlgorithm = "ecdsa"
	Ed25519PrivateKeyAlgorithm PrivateKeyAlgorithm = "ed25519"
)

const (
	UsernameSecretKey   = "username"
	PasswordSecretKey   = "password"
	CAFileSecretKey     = "caFile"
	CertFileSecretKey   = "certFile"
	KeyFileSecretKey    = "keyFile"
	PrivateKeySecretKey = "identity"
	PublicKeySecretKey  = "identity.pub"
	KnownHostsSecretKey = "known_hosts"
	BearerTokenKey      = "bearerToken"
)

type Options struct {
	Name                string
	Namespace           string
	Labels              map[string]string
	Registry            string
	SSHHostname         string
	PrivateKeyAlgorithm PrivateKeyAlgorithm
	RSAKeyBits          int
	ECDSACurve          elliptic.Curve
	Keypair             *ssh.KeyPair
	Username            string
	Password            string
	CAFile              []byte
	CertFile            []byte
	KeyFile             []byte
	TargetPath          string
	ManifestFile        string
	BearerToken         string
}

func MakeDefaultOptions() Options {
	return Options{
		Name:                "flux-system",
		Namespace:           "flux-system",
		Labels:              map[string]string{},
		PrivateKeyAlgorithm: RSAPrivateKeyAlgorithm,
		Username:            "",
		Password:            "",
		CAFile:              []byte{},
		CertFile:            []byte{},
		KeyFile:             []byte{},
		ManifestFile:        "secret.yaml",
		BearerToken:         "",
	}
}
