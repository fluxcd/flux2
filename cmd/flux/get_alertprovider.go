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
	"os"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/flux2/internal/utils"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
)

var getAlertProviderCmd = &cobra.Command{
	Use:   "alert-providers",
	Short: "Get Provider statuses",
	Long:  "The get alert-provider command prints the statuses of the resources.",
	Example: `  # List all Providers and their status
  flux get alert-providers
`,
	RunE: getAlertProviderCmdRun,
}

func init() {
	getCmd.AddCommand(getAlertProviderCmd)
}

func getAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !getArgs.allNamespaces {
		listOpts = append(listOpts, client.InNamespace(rootArgs.namespace))
	}
	var list notificationv1.ProviderList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no providers found in %s namespace", rootArgs.namespace)
		return nil
	}

	header := []string{"Name", "Ready", "Message"}
	if getArgs.allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, provider := range list.Items {
		row := []string{}
		if c := apimeta.FindStatusCondition(provider.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				provider.GetName(),
				string(c.Status),
				c.Message,
			}
		} else {
			row = []string{
				provider.GetName(),
				string(metav1.ConditionFalse),
				"waiting to be reconciled",
			}
		}
		if getArgs.allNamespaces {
			row = append([]string{provider.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.PrintTable(os.Stdout, header, rows)
	return nil
}
