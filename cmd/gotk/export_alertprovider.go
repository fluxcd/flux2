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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

var exportAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Export Provider resources in YAML format",
	Long:  "The export alert-provider command exports one or all Provider resources in YAML format.",
	Example: `  # Export all Provider resources
  gotk export alert-provider --all > alert-providers.yaml

  # Export a Provider
  gotk export alert-provider slack > slack.yaml
`,
	RunE: exportAlertProviderCmdRun,
}

func init() {
	exportCmd.AddCommand(exportAlertProviderCmd)
}

func exportAlertProviderCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if exportAll {
		var list notificationv1.ProviderList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logger.Failuref("no alertproviders found in %s namespace", namespace)
			return nil
		}

		for _, alertProvider := range list.Items {
			if err := exportAlertProvider(alertProvider); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var alertProvider notificationv1.Provider
		err = kubeClient.Get(ctx, namespacedName, &alertProvider)
		if err != nil {
			return err
		}
		return exportAlertProvider(alertProvider)
	}
	return nil
}

func exportAlertProvider(alertProvider notificationv1.Provider) error {
	gvk := notificationv1.GroupVersion.WithKind("Provider")
	export := notificationv1.Provider{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        alertProvider.Name,
			Namespace:   alertProvider.Namespace,
			Labels:      alertProvider.Labels,
			Annotations: alertProvider.Annotations,
		},
		Spec: alertProvider.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
