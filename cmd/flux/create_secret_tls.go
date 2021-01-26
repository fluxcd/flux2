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
	"io/ioutil"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"

	"github.com/fluxcd/flux2/internal/utils"
)

var createSecretTLSCmd = &cobra.Command{
	Use:   "tls [name]",
	Short: "Create or update a Kubernetes secret with TLS certificates",
	Long: `
The create secret tls command generates a Kubernetes secret with certificates for use with TLS.`,
	Example: `
  # Create a TLS secret on disk and encrypt it with Mozilla SOPS.
  # Files are expected to be PEM-encoded.
  flux create secret tls certs \
    --namespace=my-namespace \
    --cert-file=./client.crt \
    --key-file=./client.key \
    --export > certs.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place certs.yaml
`,
	RunE: createSecretTLSCmdRun,
}

type secretTLSFlags struct {
	certFile string
	keyFile  string
	caFile   string
}

var secretTLSArgs secretTLSFlags

func initSecretTLSFlags(flags *pflag.FlagSet, args *secretTLSFlags) {
	flags.StringVar(&args.certFile, "cert-file", "", "TLS authentication cert file path")
	flags.StringVar(&args.keyFile, "key-file", "", "TLS authentication key file path")
	flags.StringVar(&args.caFile, "ca-file", "", "TLS authentication CA file path")
}

func init() {
	flags := createSecretTLSCmd.Flags()
	initSecretTLSFlags(flags, &secretTLSArgs)
	createSecretCmd.AddCommand(createSecretTLSCmd)
}

func populateSecretTLS(secret *corev1.Secret, args secretTLSFlags) error {
	if args.certFile != "" && args.keyFile != "" {
		cert, err := ioutil.ReadFile(args.certFile)
		if err != nil {
			return fmt.Errorf("failed to read repository cert file '%s': %w", args.certFile, err)
		}
		secret.StringData["certFile"] = string(cert)

		key, err := ioutil.ReadFile(args.keyFile)
		if err != nil {
			return fmt.Errorf("failed to read repository key file '%s': %w", args.keyFile, err)
		}
		secret.StringData["keyFile"] = string(key)
	}

	if args.caFile != "" {
		ca, err := ioutil.ReadFile(args.caFile)
		if err != nil {
			return fmt.Errorf("failed to read repository CA file '%s': %w", args.caFile, err)
		}
		secret.StringData["caFile"] = string(ca)
	}
	return nil
}

func createSecretTLSCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("secret name is required")
	}
	name := args[0]
	secret, err := makeSecret(name)
	if err != nil {
		return err
	}

	if err = populateSecretTLS(&secret, secretTLSArgs); err != nil {
		return err
	}

	if createArgs.export {
		return exportSecret(secret)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

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
