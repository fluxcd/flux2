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
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var getAlertCmd = &cobra.Command{
	Use:     "alerts",
	Aliases: []string{"alert"},
	Short:   "Get Alert statuses",
	Long:    withPreviewNote("The get alert command prints the statuses of the resources."),
	Example: `  # List all Alerts and their status
  flux get alerts`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.AlertKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: alertType,
			list:    &alertListAdapter{&notificationv1.AlertList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*notificationv1.Alert)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v alert", obj)
			}

			sink := alertListAdapter{
				&notificationv1.AlertList{
					Items: []notificationv1.Alert{
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
	getCmd.AddCommand(getAlertCmd)
}

func (s alertListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := string(metav1.ConditionTrue), "Alert is Ready"
	return append(nameColumns(&item, includeNamespace, includeKind),
		cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (s alertListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Suspended", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s alertListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	return false
}
