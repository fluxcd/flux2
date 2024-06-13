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
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var getAlertProviderCmd = &cobra.Command{
	Use:     "alert-providers",
	Aliases: []string{"alert-provider"},
	Short:   "Get Provider statuses",
	Long:    withPreviewNote("The get alert-provider command prints the statuses of the resources."),
	Example: `  # List all Providers and their status
  flux get alert-providers`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: alertProviderType,
			list:    alertProviderListAdapter{&notificationv1.ProviderList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*notificationv1.Provider)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v alert-provider", obj)
			}

			sink := alertProviderListAdapter{
				&notificationv1.ProviderList{
					Items: []notificationv1.Provider{
						*o,
					},
				},
			}
			return sink, nil
		})

		if err != nil {
			return err
		}

		if err := get.run(cmd, args); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	getCmd.AddCommand(getAlertProviderCmd)
}

func (s alertProviderListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := string(metav1.ConditionTrue), "Provider is Ready"
	return append(nameColumns(&item, includeNamespace, includeKind), status, msg)
}

func (s alertProviderListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s alertProviderListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	return false
}
