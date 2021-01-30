/*
Copyright 2020 The Flux authors

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
	"context"
	"crypto/elliptic"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/ssh"
)

var createSecretGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Create or update a Kubernetes secret for Git authentication",
	Long: `
The create secret git command generates a Kubernetes secret with Git credentials.
For Git over SSH, the host and SSH keys are automatically generated and stored in the secret.
For Git over HTTP/S, the provided basic authentication credentials are stored in the secret.`,
	Example: `  # Create a Git SSH authentication secret using an ECDSA P-521 curve public key

  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a secret for a Git repository using basic authentication
  flux create secret git podinfo-auth \
    --url=https://github.com/stefanprodan/podinfo \
    --username=username \
    --password=password

  # Create a Git SSH secret on disk and print the deploy key
  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --export > podinfo-auth.yaml

  yq read podinfo-auth.yaml 'data."identity.pub"' | base64 --decode

  # Create a Git SSH secret on disk and encrypt it with Mozilla SOPS
  flux create secret git podinfo-auth \
    --namespace=apps \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --export > podinfo-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place podinfo-auth.yaml
`,
	RunE: createSecretGitCmdRun,
}

type secretGitFlags struct {
	url          string
	username     string
	password     string
	keyAlgorithm flags.PublicKeyAlgorithm
	rsaBits      flags.RSAKeyBits
	ecdsaCurve   flags.ECDSACurve
}

var secretGitArgs = NewSecretGitFlags()

func init() {
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.url, "url", "", "git address, e.g. ssh://git@host/org/repository")
	createSecretGitCmd.Flags().StringVarP(&secretGitArgs.username, "username", "u", "", "basic authentication username")
	createSecretGitCmd.Flags().StringVarP(&secretGitArgs.password, "password", "p", "", "basic authentication password")
	createSecretGitCmd.Flags().Var(&secretGitArgs.keyAlgorithm, "ssh-key-algorithm", secretGitArgs.keyAlgorithm.Description())
	createSecretGitCmd.Flags().Var(&secretGitArgs.rsaBits, "ssh-rsa-bits", secretGitArgs.rsaBits.Description())
	createSecretGitCmd.Flags().Var(&secretGitArgs.ecdsaCurve, "ssh-ecdsa-curve", secretGitArgs.ecdsaCurve.Description())

	createSecretCmd.AddCommand(createSecretGitCmd)
}

func NewSecretGitFlags() secretGitFlags {
	return secretGitFlags{
		keyAlgorithm: "rsa",
		rsaBits:      2048,
		ecdsaCurve:   flags.ECDSACurve{Curve: elliptic.P384()},
	}
}

func createSecretGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("secret name is required")
	}
	name := args[0]
	secret, err := makeSecret(name)
	if err != nil {
		return err
	}

	if secretGitArgs.url == "" {
		return fmt.Errorf("url is required")
	}

	u, err := url.Parse(secretGitArgs.url)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	switch u.Scheme {
	case "ssh":
		pair, err := generateKeyPair(ctx, secretGitArgs.keyAlgorithm, secretGitArgs.rsaBits, secretGitArgs.ecdsaCurve)
		if err != nil {
			return err
		}

		hostKey, err := scanHostKey(ctx, u)
		if err != nil {
			return err
		}

		secret.StringData = map[string]string{
			"identity":     string(pair.PrivateKey),
			"identity.pub": string(pair.PublicKey),
			"known_hosts":  string(hostKey),
		}

		if !createArgs.export {
			logger.Generatef("deploy key: %s", string(pair.PublicKey))
		}
	case "http", "https":
		if secretGitArgs.username == "" || secretGitArgs.password == "" {
			return fmt.Errorf("for Git over HTTP/S the username and password are required")
		}

		// TODO: add cert data when it's implemented in source-controller
		secret.StringData = map[string]string{
			"username": secretGitArgs.username,
			"password": secretGitArgs.password,
		}
	default:
		return fmt.Errorf("git URL scheme '%s' not supported, can be: ssh, http and https", u.Scheme)
	}

	if createArgs.export {
		return exportSecret(secret)
	}

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	if err := upsertSecret(ctx, kubeClient, secret); err != nil {
		return err
	}
	logger.Actionf("secret '%s' created in '%s' namespace", name, rootArgs.namespace)

	return nil
}

func generateKeyPair(ctx context.Context, alg flags.PublicKeyAlgorithm, rsa flags.RSAKeyBits, ecdsa flags.ECDSACurve) (*ssh.KeyPair, error) {
	var keyGen ssh.KeyPairGenerator
	switch algorithm := alg.String(); algorithm {
	case "rsa":
		keyGen = ssh.NewRSAGenerator(int(rsa))
	case "ecdsa":
		keyGen = ssh.NewECDSAGenerator(ecdsa.Curve)
	case "ed25519":
		keyGen = ssh.NewEd25519Generator()
	default:
		return nil, fmt.Errorf("unsupported public key algorithm: %s", algorithm)
	}
	pair, err := keyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("key pair generation failed, error: %w", err)
	}
	return pair, nil
}

func scanHostKey(ctx context.Context, url *url.URL) ([]byte, error) {
	host := url.Host
	if url.Port() == "" {
		host = host + ":22"
	}
	hostKey, err := ssh.ScanHostKey(host, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("SSH key scan for host %s failed, error: %w", host, err)
	}
	return hostKey, nil
}
