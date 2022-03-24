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

package main

import (
	"context"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
)

var createSecretHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Create or update a Kubernetes secret for Helm repository authentication",
	Long:  `The create secret helm command generates a Kubernetes secret with basic authentication credentials.`,
	Example: ` # Create a Helm authentication secret on disk and encrypt it with Mozilla SOPS
  flux create secret helm repo-auth \
    --namespace=my-namespace \
    --username=my-username \
    --password=my-password \
    --export > repo-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place repo-auth.yaml

  # Create a Helm authentication secret using a custom TLS cert
  flux create secret helm repo-auth \
    --username=username \
    --password=password \
    --cert-file=./cert.crt \
    --key-file=./key.crt \
    --ca-file=./ca.crt`,
	RunE: createSecretHelmCmdRun,
}

type secretHelmFlags struct {
	username string
	password string
	secretTLSFlags
}

var secretHelmArgs secretHelmFlags

func init() {
	createSecretHelmCmd.Flags().StringVarP(&secretHelmArgs.username, "username", "u", "", "basic authentication username")
	createSecretHelmCmd.Flags().StringVarP(&secretHelmArgs.password, "password", "p", "", "basic authentication password")
	initSecretTLSFlags(createSecretHelmCmd.Flags(), &secretHelmArgs.secretTLSFlags)
	createSecretCmd.AddCommand(createSecretHelmCmd)
}

func createSecretHelmCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	opts := sourcesecret.Options{
		Name:         name,
		Namespace:    *kubeconfigArgs.Namespace,
		Labels:       labels,
		Username:     secretHelmArgs.username,
		Password:     secretHelmArgs.password,
		CAFilePath:   secretHelmArgs.caFile,
		CertFilePath: secretHelmArgs.certFile,
		KeyFilePath:  secretHelmArgs.keyFile,
	}
	secret, err := sourcesecret.Generate(opts)
	if err != nil {
		return err
	}

	if createArgs.export {
		rootCmd.Println(secret.Content)
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

	logger.Actionf("helm secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)
	return nil
}
