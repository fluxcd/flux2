/*
Copyright 2022 The Flux authors

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

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var createSecretOCICmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Create or update a Kubernetes image pull secret",
	Long:  withPreviewNote(`The create secret oci command generates a Kubernetes secret that can be used for OCIRepository authentication`),
	Example: `  # Create an OCI authentication secret on disk and encrypt it with Mozilla SOPS
  flux create secret oci podinfo-auth \
    --url=ghcr.io \
    --username=username \
    --password=password \
    --export > repo-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place repo-auth.yaml
	`,
	RunE: createSecretOCICmdRun,
}

type secretOCIFlags struct {
	url      string
	password string
	username string
}

var secretOCIArgs = secretOCIFlags{}

func init() {
	createSecretOCICmd.Flags().StringVar(&secretOCIArgs.url, "url", "", "oci repository address e.g ghcr.io/stefanprodan/charts")
	createSecretOCICmd.Flags().StringVarP(&secretOCIArgs.username, "username", "u", "", "basic authentication username")
	createSecretOCICmd.Flags().StringVarP(&secretOCIArgs.password, "password", "p", "", "basic authentication password")

	createSecretCmd.AddCommand(createSecretOCICmd)
}

func createSecretOCICmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	secretName := args[0]

	if secretOCIArgs.url == "" {
		return fmt.Errorf("--url is required")
	}

	if secretOCIArgs.username == "" {
		return fmt.Errorf("--username is required")
	}

	if secretOCIArgs.password == "" {
		return fmt.Errorf("--password is required")
	}

	if _, err := name.ParseReference(secretOCIArgs.url); err != nil {
		return fmt.Errorf("error parsing url: '%s'", err)
	}

	opts := sourcesecret.Options{
		Name:      secretName,
		Namespace: *kubeconfigArgs.Namespace,
		Registry:  secretOCIArgs.url,
		Password:  secretOCIArgs.password,
		Username:  secretOCIArgs.username,
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

	logger.Actionf("oci secret '%s' created in '%s' namespace", secretName, *kubeconfigArgs.Namespace)
	return nil
}
