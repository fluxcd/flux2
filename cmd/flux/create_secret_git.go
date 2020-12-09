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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
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

var (
	secretGitURL          string
	secretGitUsername     string
	secretGitPassword     string
	secretGitKeyAlgorithm flags.PublicKeyAlgorithm = "rsa"
	secretGitRSABits      flags.RSAKeyBits         = 2048
	secretGitECDSACurve                            = flags.ECDSACurve{Curve: elliptic.P384()}
)

func init() {
	createSecretGitCmd.Flags().StringVar(&secretGitURL, "url", "", "git address, e.g. ssh://git@host/org/repository")
	createSecretGitCmd.Flags().StringVarP(&secretGitUsername, "username", "u", "", "basic authentication username")
	createSecretGitCmd.Flags().StringVarP(&secretGitPassword, "password", "p", "", "basic authentication password")
	createSecretGitCmd.Flags().Var(&secretGitKeyAlgorithm, "ssh-key-algorithm", sourceGitKeyAlgorithm.Description())
	createSecretGitCmd.Flags().Var(&secretGitRSABits, "ssh-rsa-bits", sourceGitRSABits.Description())
	createSecretGitCmd.Flags().Var(&secretGitECDSACurve, "ssh-ecdsa-curve", sourceGitECDSACurve.Description())

	createSecretCmd.AddCommand(createSecretGitCmd)
}

func createSecretGitCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("secret name is required")
	}
	name := args[0]

	if secretGitURL == "" {
		return fmt.Errorf("url is required")
	}

	u, err := url.Parse(secretGitURL)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	secretLabels, err := parseLabels()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    secretLabels,
		},
	}

	switch u.Scheme {
	case "ssh":
		pair, err := generateKeyPair(ctx)
		if err != nil {
			return err
		}

		hostKey, err := scanHostKey(ctx, u)
		if err != nil {
			return err
		}

		secret.Data = map[string][]byte{
			"identity":     pair.PrivateKey,
			"identity.pub": pair.PublicKey,
			"known_hosts":  hostKey,
		}

		if !export {
			logger.Generatef("deploy key: %s", string(pair.PublicKey))
		}
	case "http", "https":
		if secretGitUsername == "" || secretGitPassword == "" {
			return fmt.Errorf("for Git over HTTP/S the username and password are required")
		}

		// TODO: add cert data when it's implemented in source-controller
		secret.Data = map[string][]byte{
			"username": []byte(secretGitUsername),
			"password": []byte(secretGitPassword),
		}
	default:
		return fmt.Errorf("git URL scheme '%s' not supported, can be: ssh, http and https", u.Scheme)
	}

	if export {
		return exportSecret(secret)
	}

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	if err := upsertSecret(ctx, kubeClient, secret); err != nil {
		return err
	}
	logger.Actionf("secret '%s' created in '%s' namespace", name, namespace)

	return nil
}
