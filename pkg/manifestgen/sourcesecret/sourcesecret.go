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

	"github.com/fluxcd/pkg/ssh"

	"github.com/fluxcd/flux2/pkg/manifestgen"
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

func Generate(options Options) (*manifestgen.Manifest, error) {
	var err error

	var keypair *ssh.KeyPair
	switch {
	case options.Username != "" && options.Password != "":
		// noop
	case len(options.PrivateKeyPath) > 0:
		if keypair, err = loadKeyPair(options.PrivateKeyPath, options.Password); err != nil {
			return nil, err
		}
	case len(options.PrivateKeyAlgorithm) > 0:
		if keypair, err = generateKeyPair(options); err != nil {
			return nil, err
		}
	}

	var hostKey []byte
	if keypair != nil {
		if hostKey, err = scanHostKey(options.SSHHostname); err != nil {
			return nil, err
		}
	}

	var caFile []byte
	if options.CAFilePath != "" {
		if caFile, err = os.ReadFile(options.CAFilePath); err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
	}

	var certFile, keyFile []byte
	if options.CertFilePath != "" && options.KeyFilePath != "" {
		if certFile, err = os.ReadFile(options.CertFilePath); err != nil {
			return nil, fmt.Errorf("failed to read cert file: %w", err)
		}
		if keyFile, err = os.ReadFile(options.KeyFilePath); err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
	}

	var dockerCfgJson []byte
	if options.Registry != "" {
		dockerCfgJson, err = generateDockerConfigJson(options.Registry, options.Username, options.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to generate json for docker config: %w", err)
		}
	}

	secret := buildSecret(keypair, hostKey, caFile, certFile, keyFile, dockerCfgJson, options)
	b, err := yaml.Marshal(secret)
	if err != nil {
		return nil, err
	}

	return &manifestgen.Manifest{
		Path:    path.Join(options.TargetPath, options.Namespace, options.ManifestFile),
		Content: fmt.Sprintf("---\n%s", resourceToString(b)),
	}, nil
}

func buildSecret(keypair *ssh.KeyPair, hostKey, caFile, certFile, keyFile, dockerCfg []byte, options Options) (secret corev1.Secret) {
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

	if dockerCfg != nil {
		secret.Type = corev1.SecretTypeDockerConfigJson
		secret.StringData[corev1.DockerConfigJsonKey] = string(dockerCfg)
		return
	}

	if options.Username != "" && options.Password != "" {
		secret.StringData[UsernameSecretKey] = options.Username
		secret.StringData[PasswordSecretKey] = options.Password
	}

	if caFile != nil {
		secret.StringData[CAFileSecretKey] = string(caFile)
	}

	if certFile != nil && keyFile != nil {
		secret.StringData[CertFileSecretKey] = string(certFile)
		secret.StringData[KeyFileSecretKey] = string(keyFile)
	}

	if keypair != nil && hostKey != nil {
		secret.StringData[PrivateKeySecretKey] = string(keypair.PrivateKey)
		secret.StringData[PublicKeySecretKey] = string(keypair.PublicKey)
		secret.StringData[KnownHostsSecretKey] = string(hostKey)
		// set password if present
		if options.Password != "" {
			secret.StringData[PasswordSecretKey] = string(options.Password)
		}
	}

	return
}

func loadKeyPair(path string, password string) (*ssh.KeyPair, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open private key file: %w", err)
	}

	var ppk cryptssh.Signer
	if password != "" {
		ppk, err = cryptssh.ParsePrivateKeyWithPassphrase(b, []byte(password))
	} else {
		ppk, err = cryptssh.ParsePrivateKey(b)
	}

	if err != nil {
		return nil, err
	}

	return &ssh.KeyPair{
		PublicKey:  cryptssh.MarshalAuthorizedKey(ppk.PublicKey()),
		PrivateKey: b,
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

func scanHostKey(host string) ([]byte, error) {
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

func generateDockerConfigJson(url, username, password string) ([]byte, error) {
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
