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
	CACrtSecretKey      = "ca.crt"
	TlsCrtSecretKey     = "tls.crt"
	TlsKeySecretKey     = "tls.key"
	PrivateKeySecretKey = "identity"
	PublicKeySecretKey  = "identity.pub"
	KnownHostsSecretKey = "known_hosts"
	BearerTokenKey      = "bearerToken"

	// Depreacted: These keys are used in the generated secrets if the
	// command was invoked with the deprecated TLS flags.
	CAFileSecretKey   = "caFile"
	CertFileSecretKey = "certFile"
	KeyFileSecretKey  = "keyFile"
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
	CACrt               []byte
	TlsCrt              []byte
	TlsKey              []byte
	TargetPath          string
	ManifestFile        string
	BearerToken         string

	// Depreacted: These fields are used to store TLS data that
	// specified by the deprecated TLS flags.
	CAFile   []byte
	CertFile []byte
	KeyFile  []byte
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
