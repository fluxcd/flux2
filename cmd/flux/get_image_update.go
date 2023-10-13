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
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/runtime"

	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
)

var getImageUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Get ImageUpdateAutomation status",
	Long:  withPreviewNote("The get image update command prints the status of ImageUpdateAutomation objects."),
	Example: `  # List all image update automation object and their status
  flux get image update

 # List image update automations from all namespaces
  flux get image update --all-namespaces`,
	ValidArgsFunction: resourceNamesCompletionFunc(autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)),
	RunE: func(cmd *cobra.Command, args []string) error {
		get := getCommand{
			apiType: imageUpdateAutomationType,
			list:    &imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{}},
			funcMap: make(typeMap),
		}

		err := get.funcMap.registerCommand(get.apiType.kind, func(obj runtime.Object) (summarisable, error) {
			o, ok := obj.(*autov1.ImageUpdateAutomation)
			if !ok {
				return nil, fmt.Errorf("Impossible to cast type %#v update", obj)
			}

			sink := imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{
				Items: []autov1.ImageUpdateAutomation{
					*o,
				}}}
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
	getImageCmd.AddCommand(getImageUpdateCmd)
}

func (s imageUpdateAutomationListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	var lastRun string
	if item.Status.LastAutomationRunTime != nil {
		lastRun = item.Status.LastAutomationRunTime.Time.Format(time.RFC3339)
	}
	return append(nameColumns(&item, includeNamespace, includeKind), lastRun,
		cases.Title(language.English).String(strconv.FormatBool(item.Spec.Suspend)), status, msg)
}

func (s imageUpdateAutomationListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Last run", "Suspended", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}

func (s imageUpdateAutomationListAdapter) statusSelectorMatches(i int, conditionType, conditionStatus string) bool {
	item := s.Items[i]
	return statusMatches(conditionType, conditionStatus, item.Status.Conditions)
}
