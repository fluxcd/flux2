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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
)

var exportAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Export Alert resources in YAML format",
	Long:  "The export alert command exports one or all Alert resources in YAML format.",
	Example: `  # Export all Alert resources
  flux export alert --all > alerts.yaml

  # Export a Alert
  flux export alert main > main.yaml
`,
	RunE: exportAlertCmdRun,
}

func init() {
	exportCmd.AddCommand(exportAlertCmd)
}

func exportAlertCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if exportAll {
		var list notificationv1.AlertList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logger.Failuref("no alerts found in %s namespace", namespace)
			return nil
		}

		for _, alert := range list.Items {
			if err := exportAlert(alert); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var alert notificationv1.Alert
		err = kubeClient.Get(ctx, namespacedName, &alert)
		if err != nil {
			return err
		}
		return exportAlert(alert)
	}
	return nil
}

func exportAlert(alert notificationv1.Alert) error {
	gvk := notificationv1.GroupVersion.WithKind("Alert")
	export := notificationv1.Alert{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        alert.Name,
			Namespace:   alert.Namespace,
			Labels:      alert.Labels,
			Annotations: alert.Annotations,
		},
		Spec: alert.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
