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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
)

var exportReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Export Receiver resources in YAML format",
	Long:  `The export receiver command exports one or all Receiver resources in YAML format.`,
	Example: `  # Export all Receiver resources
  flux export receiver --all > receivers.yaml

  # Export a Receiver
  flux export receiver main > main.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind)),
	RunE: exportCommand{
		list:   receiverListAdapter{&notificationv1.ReceiverList{}},
		object: receiverAdapter{&notificationv1.Receiver{}},
	}.run,
}

func init() {
	exportCmd.AddCommand(exportReceiverCmd)
}

func exportReceiver(receiver *notificationv1.Receiver) interface{} {
	gvk := notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind)
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

	return export
}

func (ex receiverAdapter) export() interface{} {
	return exportReceiver(ex.Receiver)
}

func (ex receiverListAdapter) exportItem(i int) interface{} {
	return exportReceiver(&ex.ReceiverList.Items[i])
}
