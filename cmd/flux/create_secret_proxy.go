/*
Copyright 2024 The Flux authors

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
	"errors"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var createSecretProxyCmd = &cobra.Command{
	Use:   "proxy [name]",
	Short: "Create or update a Kubernetes secret for proxy authentication",
	Long: `The create secret proxy command generates a Kubernetes secret with the
proxy address and the basic authentication credentials.`,
	Example: ` # Create a proxy secret on disk and encrypt it with SOPS
  flux create secret proxy my-proxy \
    --namespace=my-namespace \
    --address=https://my-proxy.com \
    --username=my-username \
    --password=my-password \
    --export > proxy.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place proxy.yaml`,

	RunE: createSecretProxyCmdRun,
}

type secretProxyFlags struct {
	address  string
	username string
	password string
}

var secretProxyArgs secretProxyFlags

func init() {
	createSecretProxyCmd.Flags().StringVar(&secretProxyArgs.address, "address", "", "proxy address")
	createSecretProxyCmd.Flags().StringVarP(&secretProxyArgs.username, "username", "u", "", "basic authentication username")
	createSecretProxyCmd.Flags().StringVarP(&secretProxyArgs.password, "password", "p", "", "basic authentication password")

	createSecretCmd.AddCommand(createSecretProxyCmd)
}

func createSecretProxyCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	if secretProxyArgs.address == "" {
		return errors.New("address is required")
	}

	opts := sourcesecret.Options{
		Name:      name,
		Namespace: *kubeconfigArgs.Namespace,
		Labels:    labels,
		Address:   secretProxyArgs.address,
		Username:  secretProxyArgs.username,
		Password:  secretProxyArgs.password,
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

	logger.Actionf("proxy secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)
	return nil
}
