/*
Copyright 2020, 2021 The Flux authors

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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var createSecretTLSCmd = &cobra.Command{
	Use:   "tls [name]",
	Short: "Create or update a Kubernetes secret with TLS certificates",
	Long:  `The create secret tls command generates a Kubernetes secret with certificates for use with TLS.`,
	Example: ` # Create a TLS secret on disk and encrypt it with SOPS.
  # Files are expected to be PEM-encoded.
  flux create secret tls certs \
    --namespace=my-namespace \
    --tls-crt-file=./client.crt \
    --tls-key-file=./client.key \
    --ca-crt-file=./ca.crt \
    --export > certs.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place certs.yaml`,
	RunE: createSecretTLSCmdRun,
}

type secretTLSFlags struct {
	caCrtFile  string
	tlsKeyFile string
	tlsCrtFile string
}

var secretTLSArgs secretTLSFlags

func init() {
	createSecretTLSCmd.Flags().StringVar(&secretTLSArgs.tlsCrtFile, "tls-crt-file", "", "TLS authentication cert file path")
	createSecretTLSCmd.Flags().StringVar(&secretTLSArgs.tlsKeyFile, "tls-key-file", "", "TLS authentication key file path")
	createSecretTLSCmd.Flags().StringVar(&secretTLSArgs.caCrtFile, "ca-crt-file", "", "TLS authentication CA file path")

	createSecretCmd.AddCommand(createSecretTLSCmd)
}

func createSecretTLSCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	opts := sourcesecret.Options{
		Name:      name,
		Namespace: *kubeconfigArgs.Namespace,
		Labels:    labels,
	}

	if secretTLSArgs.caCrtFile != "" {
		opts.CACrt, err = os.ReadFile(secretTLSArgs.caCrtFile)
		if err != nil {
			return fmt.Errorf("unable to read TLS CA file: %w", err)
		}
	}

	if secretTLSArgs.tlsCrtFile != "" && secretTLSArgs.tlsKeyFile != "" {
		if opts.TLSCrt, err = os.ReadFile(secretTLSArgs.tlsCrtFile); err != nil {
			return fmt.Errorf("failed to read cert file: %w", err)
		}
		if opts.TLSKey, err = os.ReadFile(secretTLSArgs.tlsKeyFile); err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}
	}

	secret, err := sourcesecret.Generate(opts)
	if err != nil {
		return err
	}

	if createArgs.export {
		rootCmd.Print(secret.Content)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()
	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}
	var s corev1.Secret
	if err := yaml.Unmarshal([]byte(secret.Content), &s); err != nil {
		return err
	}
	if err := upsertSecret(ctx, kubeClient, s); err != nil {
		return err
	}

	logger.Actionf("tls secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)
	return nil
}
