/*
Copyright 2020 The Flux CD contributors.

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
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var createAlertProviderCmd = &cobra.Command{
	Use:     "alert-provider [name]",
	Aliases: []string{"ap"},
	Short:   "Create or update a Provider resource",
	Long:    "The create alert-provider command generates a Provider resource.",
	Example: `  # Create a Provider for a Slack channel
  gotk create ap slack \
  --type slack \
  --channel general \
  --address https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK \
  --secret-ref webhook-url

  # Create a Provider for a Github repository
  gotk create ap github-podinfo \
  --type github \
  --address https://github.com/stefanprodan/podinfo \
  --secret-ref github-token
`,
	RunE: createAlertProviderCmdRun,
}

var (
	apType      string
	apChannel   string
	apUsername  string
	apAddress   string
	apSecretRef string
)

func init() {
	createAlertProviderCmd.Flags().StringVar(&apType, "type", "", "type of provider")
	createAlertProviderCmd.Flags().StringVar(&apChannel, "channel", "", "channel to send messages to in the case of a chat provider")
	createAlertProviderCmd.Flags().StringVar(&apUsername, "username", "", "bot username used by the provider")
	createAlertProviderCmd.Flags().StringVar(&apAddress, "address", "", "path to either the git repository, chat provider or webhook")
	createAlertProviderCmd.Flags().StringVar(&apSecretRef, "secret-ref", "", "name of secret containing authentication token")
	createCmd.AddCommand(createAlertProviderCmd)
}

func createAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("provider name is required")
	}
	name := args[0]

	if apType == "" {
		return fmt.Errorf("type is required")
	}

	sourceLabels, err := parseLabels()
	if err != nil {
		return err
	}

	if !export {
		logger.Generatef("generating provider")
	}

	alertProvider := notificationv1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    sourceLabels,
		},
		Spec: notificationv1.ProviderSpec{
			Type:     apType,
			Channel:  apChannel,
			Username: apUsername,
			Address:  apAddress,
			SecretRef: &corev1.LocalObjectReference{
				Name: apSecretRef,
			},
		},
	}

	if export {
		return exportAlertProvider(alertProvider)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	logger.Actionf("applying provider")
	if err := upsertAlertProvider(ctx, kubeClient, alertProvider); err != nil {
		return err
	}

	logger.Waitingf("waiting for reconciliation")
	if err := wait.PollImmediate(pollInterval, timeout,
		isAlertProviderReady(ctx, kubeClient, name, namespace)); err != nil {
		return err
	}

	logger.Successf("provider %s is ready", name)

	return nil
}

func upsertAlertProvider(ctx context.Context, kubeClient client.Client, alertProvider notificationv1.Provider) error {
	namespacedName := types.NamespacedName{
		Namespace: alertProvider.GetNamespace(),
		Name:      alertProvider.GetName(),
	}

	var existing notificationv1.Provider
	err := kubeClient.Get(ctx, namespacedName, &existing)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := kubeClient.Create(ctx, &alertProvider); err != nil {
				return err
			} else {
				logger.Successf("provider created")
				return nil
			}
		}
		return err
	}

	existing.Labels = alertProvider.Labels
	existing.Spec = alertProvider.Spec
	if err := kubeClient.Update(ctx, &existing); err != nil {
		return err
	}

	logger.Successf("provider updated")
	return nil
}

func isAlertProviderReady(ctx context.Context, kubeClient client.Client, name, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		var alertProvider notificationv1.Provider
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		err := kubeClient.Get(ctx, namespacedName, &alertProvider)
		if err != nil {
			return false, err
		}

		if c := meta.GetCondition(alertProvider.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				return true, nil
			case corev1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}
