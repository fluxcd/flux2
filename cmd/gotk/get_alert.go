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
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
)

var getAlertCmd = &cobra.Command{
	Use:     "alert",
	Aliases: []string{},
	Short:   "Get Alert statuses",
	Long:    "The get alert command prints the statuses of the resources.",
	Example: `  # List all Alerts and their status
  gotk get alert
`,
	RunE: getAlertCmdRun,
}

func init() {
	getCmd.AddCommand(getAlertCmd)
}

func getAlertCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !allNamespaces {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}
	var list notificationv1.AlertList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no alerts found in %s namespace", namespace)
		return nil
	}

	header := []string{"Name", "Suspended", "Ready", "Message"}
	if allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, alert := range list.Items {
		row := []string{}
		if c := meta.GetCondition(alert.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				alert.GetName(),
				//alert.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(alert.Spec.Suspend)),
				string(c.Status),
				c.Message,
			}
		} else {
			row = []string{
				alert.GetName(),
				//alert.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(alert.Spec.Suspend)),
				string(corev1.ConditionFalse),
				"waiting to be reconciled",
			}
		}
		if allNamespaces {
			row = append([]string{alert.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.printTable(os.Stdout, header, rows)
	return nil
}
