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
	"fmt"
	"os"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var createSecretGitHubAppCmd = &cobra.Command{
	Use:   "githubapp [name]",
	Short: "Create or update a github app secret",
	Long:  withPreviewNote(`The create secret githubapp command generates a Kubernetes secret that can be used for GitRepository authentication with github app`),
	Example: `  # Create a githubapp authentication secret on disk and encrypt it with Mozilla SOPS
  flux create secret githubapp podinfo-auth \
    --app-id="1" \
    --app-installation-id="2" \
    --app-private-key=./private-key-file.pem \
    --export > githubapp-auth.yaml

  sops --encrypt --encrypted-regex '^(data|stringData)$' \
    --in-place githubapp-auth.yaml
	`,
	RunE: createSecretGitHubAppCmdRun,
}

type secretGitHubAppFlags struct {
	appID             string
	appInstallationID string
	privateKeyFile    string
	baseURL           string
}

var secretGitHubAppArgs = secretGitHubAppFlags{}

func init() {
	createSecretGitHubAppCmd.Flags().StringVar(&secretGitHubAppArgs.appID, "app-id", "", "github app ID")
	createSecretGitHubAppCmd.Flags().StringVar(&secretGitHubAppArgs.appInstallationID, "app-installation-id", "", "github app installation ID")
	createSecretGitHubAppCmd.Flags().StringVar(&secretGitHubAppArgs.privateKeyFile, "app-private-key", "", "github app private key file path")
	createSecretGitHubAppCmd.Flags().StringVar(&secretGitHubAppArgs.baseURL, "app-base-url", "", "github app base URL")

	createSecretCmd.AddCommand(createSecretGitHubAppCmd)
}

func createSecretGitHubAppCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	secretName := args[0]

	if secretGitHubAppArgs.appID == "" {
		return fmt.Errorf("--app-id is required")
	}

	if secretGitHubAppArgs.appInstallationID == "" {
		return fmt.Errorf("--app-installation-id is required")
	}

	if secretGitHubAppArgs.privateKeyFile == "" {
		return fmt.Errorf("--app-private-key is required")
	}

	privateKey, err := os.ReadFile(secretGitHubAppArgs.privateKeyFile)
	if err != nil {
		return fmt.Errorf("unable to read private key file: %w", err)
	}

	opts := sourcesecret.Options{
		Name:                    secretName,
		Namespace:               *kubeconfigArgs.Namespace,
		GitHubAppID:             secretGitHubAppArgs.appID,
		GitHubAppInstallationID: secretGitHubAppArgs.appInstallationID,
		GitHubAppPrivateKey:     string(privateKey),
	}

	if secretGitHubAppArgs.baseURL != "" {
		opts.GitHubAppBaseURL = secretGitHubAppArgs.baseURL
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

	logger.Actionf("githubapp secret '%s' created in '%s' namespace", secretName, *kubeconfigArgs.Namespace)
	return nil
}
