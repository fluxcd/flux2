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
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var createSecretGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Create or update a Kubernetes secret for Git authentication",
	Long: `The create secret git command generates a Kubernetes secret with Git credentials.
For Git over SSH, the host and SSH keys are automatically generated and stored
in the secret.
For Git over HTTP/S, the provided basic authentication credentials or bearer
authentication token are stored in the secret.`,
	Example: `  # Create a Git SSH authentication secret using an ECDSA P-521 curve public key

  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --ssh-key-algorithm=ecdsa \
    --ssh-ecdsa-curve=p521

  # Create a Git SSH authentication secret with a passwordless private key from file
  # The public SSH host key will still be gathered from the host
  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --private-key-file=./private.key

  # Create a Git SSH authentication secret with a passworded private key from file
  # The public SSH host key will still be gathered from the host
  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --private-key-file=./private.key \
    --password=<password>

  # Create a secret for a Git repository using basic authentication
  flux create secret git podinfo-auth \
    --url=https://github.com/stefanprodan/podinfo \
    --username=username \
    --password=password

  # Create a Git SSH secret on disk
  flux create secret git podinfo-auth \
    --url=ssh://git@github.com/stefanprodan/podinfo \
    --export > podinfo-auth.yaml

  # Print the deploy key
  yq eval '.stringData."identity.pub"' podinfo-auth.yaml

  # Encrypt the secret on disk with Mozilla SOPS
  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place podinfo-auth.yaml`,
	RunE: createSecretGitCmdRun,
}

type secretGitFlags struct {
	url            string
	username       string
	password       string
	keyAlgorithm   flags.PublicKeyAlgorithm
	rsaBits        flags.RSAKeyBits
	ecdsaCurve     flags.ECDSACurve
	caFile         string
	caCrtFile      string
	privateKeyFile string
	bearerToken    string
}

var secretGitArgs = NewSecretGitFlags()

func init() {
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.url, "url", "", "git address, e.g. ssh://git@host/org/repository")
	createSecretGitCmd.Flags().StringVarP(&secretGitArgs.username, "username", "u", "", "basic authentication username")
	createSecretGitCmd.Flags().StringVarP(&secretGitArgs.password, "password", "p", "", "basic authentication password")
	createSecretGitCmd.Flags().Var(&secretGitArgs.keyAlgorithm, "ssh-key-algorithm", secretGitArgs.keyAlgorithm.Description())
	createSecretGitCmd.Flags().Var(&secretGitArgs.rsaBits, "ssh-rsa-bits", secretGitArgs.rsaBits.Description())
	createSecretGitCmd.Flags().Var(&secretGitArgs.ecdsaCurve, "ssh-ecdsa-curve", secretGitArgs.ecdsaCurve.Description())
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.caFile, "ca-file", "", "path to TLS CA file used for validating self-signed certificates")
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.caCrtFile, "ca-crt-file", "", "path to TLS CA certificate file used for validating self-signed certificates; takes precedence over --ca-file")
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.privateKeyFile, "private-key-file", "", "path to a passwordless private key file used for authenticating to the Git SSH server")
	createSecretGitCmd.Flags().StringVar(&secretGitArgs.bearerToken, "bearer-token", "", "bearer authentication token")

	createSecretCmd.AddCommand(createSecretGitCmd)
}

func NewSecretGitFlags() secretGitFlags {
	return secretGitFlags{
		keyAlgorithm: flags.PublicKeyAlgorithm(sourcesecret.ECDSAPrivateKeyAlgorithm),
		rsaBits:      2048,
		ecdsaCurve:   flags.ECDSACurve{Curve: elliptic.P384()},
	}
}

func createSecretGitCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	if secretGitArgs.url == "" {
		return fmt.Errorf("url is required")
	}

	u, err := url.Parse(secretGitArgs.url)
	if err != nil {
		return fmt.Errorf("git URL parse failed: %w", err)
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	opts := sourcesecret.Options{
		Name:         name,
		Namespace:    *kubeconfigArgs.Namespace,
		Labels:       labels,
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	switch u.Scheme {
	case "ssh":
		keypair, err := sourcesecret.LoadKeyPairFromPath(secretGitArgs.privateKeyFile, secretGitArgs.password)
		if err != nil {
			return err
		}
		opts.Keypair = keypair
		opts.SSHHostname = u.Host
		opts.PrivateKeyAlgorithm = sourcesecret.PrivateKeyAlgorithm(secretGitArgs.keyAlgorithm)
		opts.RSAKeyBits = int(secretGitArgs.rsaBits)
		opts.ECDSACurve = secretGitArgs.ecdsaCurve.Curve
		opts.Password = secretGitArgs.password
	case "http", "https":
		if (secretGitArgs.username == "" || secretGitArgs.password == "") && secretGitArgs.bearerToken == "" {
			return fmt.Errorf("for Git over HTTP/S the username and password, or a bearer token is required")
		}
		opts.Username = secretGitArgs.username
		opts.Password = secretGitArgs.password
		opts.BearerToken = secretGitArgs.bearerToken
		if secretGitArgs.username != "" && secretGitArgs.password != "" && secretGitArgs.bearerToken != "" {
			return fmt.Errorf("user credentials and bearer token cannot be used together")
		}

		// --ca-crt-file takes precedence over --ca-file.
		if secretGitArgs.caCrtFile != "" {
			opts.CACrt, err = os.ReadFile(secretGitArgs.caCrtFile)
			if err != nil {
				return fmt.Errorf("unable to read TLS CA file: %w", err)
			}
		} else if secretGitArgs.caFile != "" {
			opts.CAFile, err = os.ReadFile(secretGitArgs.caFile)
			if err != nil {
				return fmt.Errorf("unable to read TLS CA file: %w", err)
			}
		}
	default:
		return fmt.Errorf("git URL scheme '%s' not supported, can be: ssh, http and https", u.Scheme)
	}

	secret, err := sourcesecret.Generate(opts)
	if err != nil {
		return err
	}

	if createArgs.export {
		rootCmd.Println(secret.Content)
		return nil
	}

	var s corev1.Secret
	if err := yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
		return err
	}

	if ppk, ok := s.StringData[sourcesecret.PublicKeySecretKey]; ok {
		logger.Generatef("deploy key: %s", ppk)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}
	if err := upsertSecret(ctx, kubeClient, s); err != nil {
		return err
	}
	logger.Actionf("git secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)

	return nil
}
