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

var exportAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Export Provider resources in YAML format",
	Long:  withPreviewNote("The export alert-provider command exports one or all Provider resources in YAML format."),
	Example: `  # Export all Provider resources
  flux export alert-provider --all > alert-providers.yaml

  # Export a Provider
  flux export alert-provider slack > slack.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: exportCommand{
		object: alertProviderAdapter{&notificationv1.Provider{}},
		list:   alertProviderListAdapter{&notificationv1.ProviderList{}},
	}.run,
}

func init() {
	exportCmd.AddCommand(exportAlertProviderCmd)
}

func exportAlertProvider(alertProvider *notificationv1.Provider) interface{} {
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
	return export
}

func (ex alertProviderAdapter) export() interface{} {
	return exportAlertProvider(ex.Provider)
}

func (ex alertProviderListAdapter) exportItem(i int) interface{} {
	return exportAlertProvider(&ex.ProviderList.Items[i])
}
