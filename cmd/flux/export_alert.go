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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var exportAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Export Alert resources in YAML format",
	Long:  withPreviewNote("The export alert command exports one or all Alert resources in YAML format."),
	Example: `  # Export all Alert resources
  flux export alert --all > alerts.yaml

  # Export a Alert
  flux export alert main > main.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.AlertKind)),
	RunE: exportCommand{
		object: alertAdapter{&notificationv1.Alert{}},
		list:   alertListAdapter{&notificationv1.AlertList{}},
	}.run,
}

func init() {
	exportCmd.AddCommand(exportAlertCmd)
}

func exportAlert(alert *notificationv1.Alert) interface{} {
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

	return export
}

func (ex alertAdapter) export() interface{} {
	return exportAlert(ex.Alert)
}

func (ex alertListAdapter) exportItem(i int) interface{} {
	return exportAlert(&ex.AlertList.Items[i])
}
