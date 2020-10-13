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

var exportReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Export Receiver resources in YAML format",
	Long:  "The export receiver command exports one or all Receiver resources in YAML format.",
	Example: `  # Export all Receiver resources
  gotk export receiver --all > receivers.yaml

  # Export a Receiver
  gotk export receiver main > main.yaml
`,
	RunE: exportReceiverCmdRun,
}

func init() {
	exportCmd.AddCommand(exportReceiverCmd)
}

func exportReceiverCmdRun(cmd *cobra.Command, args []string) error {
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
			if err := exportReceiver(receiver); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var receiver notificationv1.Receiver
		err = kubeClient.Get(ctx, namespacedName, &receiver)
		if err != nil {
			return err
		}
		return exportReceiver(receiver)
	}
	return nil
}

func exportReceiver(receiver notificationv1.Receiver) error {
	gvk := notificationv1.GroupVersion.WithKind("Receiver")
	export := notificationv1.Receiver{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        receiver.Name,
			Namespace:   receiver.Namespace,
			Labels:      receiver.Labels,
			Annotations: receiver.Annotations,
		},
		Spec: receiver.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
