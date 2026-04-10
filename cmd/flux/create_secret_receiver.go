/*
Copyright 2026 The Flux authors

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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
)

var createSecretReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Create or update a Kubernetes secret for a Receiver webhook",
	Long: `The create secret receiver command generates a Kubernetes secret with
the token used for webhook payload validation and an annotation with the
computed webhook URL.`,
	Example: `  # Create a receiver secret for a GitHub webhook
  flux create secret receiver github-receiver \
    --namespace=my-namespace \
    --type=github \
    --hostname=flux.example.com \
    --export

  # Create a receiver secret for GCR with email claim
  flux create secret receiver gcr-receiver \
    --namespace=my-namespace \
    --type=gcr \
    --hostname=flux.example.com \
    --email-claim=sa@project.iam.gserviceaccount.com \
    --export`,
	RunE: createSecretReceiverCmdRun,
}

type secretReceiverFlags struct {
	receiverType flags.ReceiverType
	token        string
	hostname     string
	emailClaim   string
}

var secretReceiverArgs secretReceiverFlags

func init() {
	createSecretReceiverCmd.Flags().Var(&secretReceiverArgs.receiverType, "type", secretReceiverArgs.receiverType.Description())
	createSecretReceiverCmd.Flags().StringVar(&secretReceiverArgs.token, "token", "", "webhook token used for payload validation and URL computation, auto-generated if not specified")
	createSecretReceiverCmd.Flags().StringVar(&secretReceiverArgs.hostname, "hostname", "", "hostname for the webhook URL e.g. flux.example.com")
	createSecretReceiverCmd.Flags().StringVar(&secretReceiverArgs.emailClaim, "email-claim", "", "IAM service account email, required for gcr type")

	createSecretCmd.AddCommand(createSecretReceiverCmd)
}

func createSecretReceiverCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if secretReceiverArgs.receiverType == "" {
		return fmt.Errorf("--type is required")
	}

	if secretReceiverArgs.hostname == "" {
		return fmt.Errorf("--hostname is required")
	}

	if secretReceiverArgs.receiverType.String() == notificationv1.GCRReceiver && secretReceiverArgs.emailClaim == "" {
		return fmt.Errorf("--email-claim is required for gcr receiver type")
	}

	labels, err := parseLabels()
	if err != nil {
		return err
	}

	opts := sourcesecret.Options{
		Name:         name,
		Namespace:    *kubeconfigArgs.Namespace,
		Labels:       labels,
		ReceiverType: secretReceiverArgs.receiverType.String(),
		Token:        secretReceiverArgs.token,
		Hostname:     secretReceiverArgs.hostname,
		EmailClaim:   secretReceiverArgs.emailClaim,
	}

	secret, err := sourcesecret.GenerateReceiver(opts)
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

	logger.Actionf("receiver secret '%s' created in '%s' namespace", name, *kubeconfigArgs.Namespace)
	return nil
}
