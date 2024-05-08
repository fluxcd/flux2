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
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/fluxcd/flux2/v2/internal/utils"
)

var createAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Create or update a Provider resource",
	Long:  withPreviewNote(`The create alert-provider command generates a Provider resource.`),
	Example: `  # Create a Provider for a Slack channel
  flux create alert-provider slack \
  --type slack \
  --channel general \
  --address https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK \
  --secret-ref webhook-url

  # Create a Provider for a Github repository
  flux create alert-provider github-podinfo \
  --type github \
  --address https://github.com/stefanprodan/podinfo \
  --secret-ref github-token`,
	RunE: createAlertProviderCmdRun,
}

type alertProviderFlags struct {
	alertType string
	channel   string
	username  string
	address   string
	secretRef string
}

var alertProviderArgs alertProviderFlags

func init() {
	createAlertProviderCmd.Flags().StringVar(&alertProviderArgs.alertType, "type", "", "type of provider")
	createAlertProviderCmd.Flags().StringVar(&alertProviderArgs.channel, "channel", "", "channel to send messages to in the case of a chat provider")
	createAlertProviderCmd.Flags().StringVar(&alertProviderArgs.username, "username", "", "bot username used by the provider")
	createAlertProviderCmd.Flags().StringVar(&alertProviderArgs.address, "address", "", "path to either the git repository, chat provider or webhook")
	createAlertProviderCmd.Flags().StringVar(&alertProviderArgs.secretRef, "secret-ref", "", "name of secret containing authentication token")
	createCmd.AddCommand(createAlertProviderCmd)
}

func createAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	if alertProviderArgs.alertType == "" {
		return fmt.Errorf("Provider type is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !createArgs.export {
		logger.Generatef("generating Provider")
	}

	provider := notificationv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: *kubeconfigArgs.Namespace,
			Labels:    sourceLabels,
		},
		Spec: notificationv1.ProviderSpec{
			Type:     alertProviderArgs.alertType,
			Channel:  alertProviderArgs.channel,
			Username: alertProviderArgs.username,
			Address:  alertProviderArgs.address,
		},
	}

	if alertProviderArgs.secretRef != "" {
		provider.Spec.SecretRef = &meta.LocalObjectReference{
			Name: alertProviderArgs.secretRef,
		}
	}

	if createArgs.export {
		return printExport(exportAlertProvider(&provider))
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	logger.Actionf("applying Provider")
	namespacedName, err := upsertAlertProvider(ctx, kubeClient, &provider)
	if err != nil {
		return err
	}

	logger.Waitingf("waiting for Provider reconciliation")
	if err := wait.PollUntilContextTimeout(ctx, rootArgs.pollInterval, rootArgs.timeout, true,
		isStaticObjectReadyConditionFunc(kubeClient, namespacedName, &provider)); err != nil {
		return err
	}

	logger.Successf("Provider %s is ready", name)

	return nil
}

func upsertAlertProvider(ctx context.Context, kubeClient client.Client,
	provider *notificationv1.Provider) (types.NamespacedName, error) {
	namespacedName := types.NamespacedName{
		Namespace: provider.GetNamespace(),
		Name:      provider.GetName(),
	}

	var existing notificationv1.Provider
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, provider); err != nil {
				return namespacedName, err
			} else {
				logger.Successf("Provider created")
				return namespacedName, nil
			}
		}
		return namespacedName, err
	}

	existing.Labels = provider.Labels
	existing.Spec = provider.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return namespacedName, err
	}
	provider = &existing
	logger.Successf("Provider updated")
	return namespacedName, nil
}
