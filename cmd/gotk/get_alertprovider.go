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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
)

var getAlertProviderCmd = &cobra.Command{
	Use:     "alert-provider",
	Aliases: []string{"ap"},
	Short:   "Get Provider statuses",
	Long:    "The get alert-provider command prints the statuses of the resources.",
	Example: `  # List all Providers and their status
  gotk get alert-provider
`,
	RunE: getAlertProviderCmdRun,
}

func init() {
	getCmd.AddCommand(getAlertProviderCmd)
}

func getAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list notificationv1.ProviderList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no providers found in %s namespace", namespace)
		return nil
	}

	for _, provider := range list.Items {
		isInitialized := false
		if c := meta.GetCondition(provider.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				logger.Successf("%s is ready", provider.GetName())
			case corev1.ConditionUnknown:
				logger.Successf("%s reconciling", provider.GetName())
			default:
				logger.Failuref("%s %s", provider.GetName(), c.Message)
			}
			isInitialized = true
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", provider.GetName())
		}
	}
	return nil
}
