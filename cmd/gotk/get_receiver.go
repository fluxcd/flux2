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

var getReceiverCmd = &cobra.Command{
	Use:     "receiver",
	Aliases: []string{"rcv"},
	Short:   "Get Receiver statuses",
	Long:    "The get receiver command prints the statuses of the resources.",
	Example: `  # List all Receiver and their status
  gotk get receiver
`,
	RunE: getReceiverCmdRun,
}

func init() {
	getCmd.AddCommand(getReceiverCmd)
}

func getReceiverCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var list notificationv1.ReceiverList
	err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no receivers found in %s namespace", namespace)
		return nil
	}

	for _, receiver := range list.Items {
		if receiver.Spec.Suspend {
			logger.Successf("%s is suspended", receiver.GetName())
			continue
		}
		isInitialized := false
		if c := meta.GetCondition(receiver.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case corev1.ConditionTrue:
				logger.Successf("%s is ready", receiver.GetName())
			case corev1.ConditionUnknown:
				logger.Successf("%s reconciling", receiver.GetName())
			default:
				logger.Failuref("%s %s", receiver.GetName(), c.Message)
			}
			isInitialized = true
		}
		if !isInitialized {
			logger.Failuref("%s is not ready", receiver.GetName())
		}
	}
	return nil
}
