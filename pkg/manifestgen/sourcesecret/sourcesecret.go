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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	cryptssh "golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/pkg/runtime/secrets"
	"github.com/fluxcd/pkg/ssh"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
)

const defaultSSHPort = 22

// types gotten from https://github.com/kubernetes/kubectl/blob/master/pkg/cmd/create/create_secret_docker.go#L64-L84

// DockerConfigJSON represents a local docker auth config file
// for pulling images.
type DockerConfigJSON struct {
	Auths DockerConfig `json:"auths"`
}

// DockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type DockerConfig map[string]DockerConfigEntry

// DockerConfigEntry holds the user information that grant the access to docker registry
type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

func GenerateGit(options Options) (*manifestgen.Manifest, error) {
	var err error

	var keypair *ssh.KeyPair
	switch {
	case options.Username != "" && options.Password != "":
		// noop
	case options.Keypair != nil:
		keypair = options.Keypair
	case len(options.PrivateKeyAlgorithm) > 0:
		if keypair, err = generateKeyPair(options); err != nil {
			return nil, err
		}
	}

	var hostKey []byte
	if keypair != nil {
		if hostKey, err = ScanHostKey(options.SSHHostname); err != nil {
			return nil, err
		}
	}

	secret := buildGitSecret(keypair, hostKey, options)
	return secretToManifest(&secret, options)
}

func GenerateTLS(options Options) (*manifestgen.Manifest, error) {
	var opts []secrets.TLSSecretOption

	if len(options.TLSCrt) > 0 || len(options.TLSKey) > 0 {
		opts = append(opts, secrets.WithCertKeyPair(options.TLSCrt, options.TLSKey))
	}
	if len(options.CACrt) > 0 {
		opts = append(opts, secrets.WithCAData(options.CACrt))
	}

	secret, err := secrets.MakeTLSSecret(options.Name, options.Namespace, opts...)
	if err != nil {
		return nil, err
	}

	secret.Labels = options.Labels
	return secretToManifest(secret, options)
}

func GenerateOCI(options Options) (*manifestgen.Manifest, error) {
	secret, err := secrets.MakeRegistrySecret(
		options.Name,
		options.Namespace,
		options.Registry,
		options.Username,
		options.Password,
	)
	if err != nil {
		return nil, err
	}

	secret.Labels = options.Labels
	return secretToManifest(secret, options)
}

func GenerateHelm(options Options) (*manifestgen.Manifest, error) {
	hasBasicAuth := options.Username != "" || options.Password != ""
	hasClientCert := len(options.TLSCrt) > 0 || len(options.TLSKey) > 0
	hasCACert := len(options.CACrt) > 0

	var secret *corev1.Secret
	var err error

	switch {
	case hasClientCert:
		// Priority 1: Client certificate (mTLS) - highest priority like CertSecretRef
		var opts []secrets.TLSSecretOption
		opts = append(opts, secrets.WithCertKeyPair(options.TLSCrt, options.TLSKey))
		if hasCACert {
			opts = append(opts, secrets.WithCAData(options.CACrt))
		}

		secret, err = secrets.MakeTLSSecret(options.Name, options.Namespace, opts...)
		if err != nil {
			return nil, err
		}

	case hasBasicAuth:
		// Priority 2: Basic authentication (can include CA certificate)
		secret, err = secrets.MakeBasicAuthSecret(
			options.Name,
			options.Namespace,
			options.Username,
			options.Password,
		)
		if err != nil {
			return nil, err
		}

		// Add CA certificate to BasicAuth secret for HTTPS repositories with custom CA
		// (e.g., self-signed certificates or internal certificate authorities)
		if hasCACert {
			if secret.StringData == nil {
				secret.StringData = make(map[string]string)
			}
			secret.StringData[CACrtSecretKey] = string(options.CACrt)
		}

	case hasCACert:
		// Priority 3: CA certificate only
		var opts []secrets.TLSSecretOption
		opts = append(opts, secrets.WithCAData(options.CACrt))

		secret, err = secrets.MakeTLSSecret(options.Name, options.Namespace, opts...)
		if err != nil {
			return nil, err
		}

	default:
		// No authentication credentials provided - create empty secret for backward compatibility
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      options.Name,
				Namespace: options.Namespace,
			},
			StringData: map[string]string{},
		}
	}

	secret.Labels = options.Labels
	return secretToManifest(secret, options)
}

func GenerateProxy(options Options) (*manifestgen.Manifest, error) {
	secret, err := secrets.MakeProxySecret(
		options.Name,
		options.Namespace,
		options.Address,
		options.Username,
		options.Password,
	)
	if err != nil {
		return nil, err
	}

	secret.Labels = options.Labels
	return secretToManifest(secret, options)
}

func GenerateNotation(options Options) (*manifestgen.Manifest, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.Name,
			Namespace: options.Namespace,
			Labels:    options.Labels,
		},
		StringData: map[string]string{},
	}

	for _, crt := range options.VerificationCrts {
		secret.StringData[crt.Name] = string(crt.CACrt)
	}

	if len(options.TrustPolicy) > 0 {
		secret.StringData[TrustPolicyKey] = string(options.TrustPolicy)
	}

	return secretToManifest(secret, options)
}

func GenerateGitHubApp(options Options) (*manifestgen.Manifest, error) {
	var opts []secrets.GitHubAppOption
	if owner := options.GitHubAppInstallationOwner; owner != "" {
		opts = append(opts, secrets.WithGitHubAppInstallationOwner(owner))
	}
	if id := options.GitHubAppInstallationID; id != "" {
		opts = append(opts, secrets.WithGitHubAppInstallationID(id))
	}
	if u := options.GitHubAppBaseURL; u != "" {
		opts = append(opts, secrets.WithGitHubAppBaseURL(u))
	}
	secret, err := secrets.MakeGitHubAppSecret(
		options.Name,
		options.Namespace,
		options.GitHubAppID,
		options.GitHubAppPrivateKey,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	secret.Labels = options.Labels
	return secretToManifest(secret, options)
}

func LoadKeyPairFromPath(path, password string) (*ssh.KeyPair, error) {
	if path == "" {
		return nil, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open private key file: %w", err)
	}
	return LoadKeyPair(b, password)
}

func LoadKeyPair(privateKey []byte, password string) (*ssh.KeyPair, error) {
	var ppk cryptssh.Signer
	var err error
	if password != "" {
		ppk, err = cryptssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(password))
	} else {
		ppk, err = cryptssh.ParsePrivateKey(privateKey)
	}
	if err != nil {
		return nil, err
	}
	return &ssh.KeyPair{
		PublicKey:  cryptssh.MarshalAuthorizedKey(ppk.PublicKey()),
		PrivateKey: privateKey,
	}, nil
}

func buildGitSecret(keypair *ssh.KeyPair, hostKey []byte, options Options) (secret corev1.Secret) {
	secret.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Secret",
	}
	secret.ObjectMeta = metav1.ObjectMeta{
		Name:      options.Name,
		Namespace: options.Namespace,
	}
	secret.Labels = options.Labels
	secret.StringData = map[string]string{}

	if options.Username != "" && options.Password != "" {
		secret.StringData[UsernameSecretKey] = options.Username
		secret.StringData[PasswordSecretKey] = options.Password
	}
	if options.BearerToken != "" {
		secret.StringData[BearerTokenKey] = options.BearerToken
	}

	if len(options.CACrt) != 0 {
		secret.StringData[CACrtSecretKey] = string(options.CACrt)
	}

	// SSH keypair (identity + identity.pub + known_hosts)
	if keypair != nil && len(hostKey) != 0 {
		secret.StringData[PrivateKeySecretKey] = string(keypair.PrivateKey)
		secret.StringData[PublicKeySecretKey] = string(keypair.PublicKey)
		secret.StringData[KnownHostsSecretKey] = string(hostKey)
		// set password if present
		if options.Password != "" {
			secret.StringData[PasswordSecretKey] = string(options.Password)
		}
	}

	return secret
}

func secretToManifest(secret *corev1.Secret, options Options) (*manifestgen.Manifest, error) {
	b, err := yaml.Marshal(secret)
	if err != nil {
		return nil, err
	}
	return &manifestgen.Manifest{
		Path:    path.Join(options.TargetPath, options.Namespace, options.ManifestFile),
		Content: fmt.Sprintf("---\n%s", resourceToString(b)),
	}, nil
}

func generateKeyPair(options Options) (*ssh.KeyPair, error) {
	var keyGen ssh.KeyPairGenerator
	switch options.PrivateKeyAlgorithm {
	case RSAPrivateKeyAlgorithm:
		keyGen = ssh.NewRSAGenerator(options.RSAKeyBits)
	case ECDSAPrivateKeyAlgorithm:
		keyGen = ssh.NewECDSAGenerator(options.ECDSACurve)
	case Ed25519PrivateKeyAlgorithm:
		keyGen = ssh.NewEd25519Generator()
	default:
		return nil, fmt.Errorf("unsupported public key algorithm: %s", options.PrivateKeyAlgorithm)
	}
	pair, err := keyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("key pair generation failed, error: %w", err)
	}
	return pair, nil
}

func ScanHostKey(host string) ([]byte, error) {
	if _, _, err := net.SplitHostPort(host); err != nil {
		// Assume we are dealing with a hostname without a port,
		// append the default SSH port as this is required for
		// host key scanning to work.
		host = fmt.Sprintf("%s:%d", host, defaultSSHPort)
	}
	hostKey, err := ssh.ScanHostKey(host, 30*time.Second, []string{}, false)
	if err != nil {
		return nil, fmt.Errorf("SSH key scan for host %s failed, error: %w", host, err)
	}
	return bytes.TrimSpace(hostKey), nil
}

func resourceToString(data []byte) string {
	data = bytes.Replace(data, []byte("  creationTimestamp: null\n"), []byte(""), 1)
	data = bytes.Replace(data, []byte("status: {}\n"), []byte(""), 1)
	return string(data)
}

func GenerateDockerConfigJson(url, username, password string) ([]byte, error) {
	cred := fmt.Sprintf("%s:%s", username, password)
	auth := base64.StdEncoding.EncodeToString([]byte(cred))
	cfg := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{
			url: {
				Username: username,
				Password: password,
				Auth:     auth,
			},
		},
	}

	return json.Marshal(cfg)
}
