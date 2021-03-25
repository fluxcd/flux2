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
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	"github.com/spf13/cobra"
)

var getAlertProviderCmd = &cobra.Command{
	Use:     "alert-providers",
	Aliases: []string{"alert-provider"},
	Short:   "Get Provider statuses",
	Long:    "The get alert-provider command prints the statuses of the resources.",
	Example: `  # List all Providers and their status
  flux get alert-providers
`,
	RunE: getCommand{
		apiType: alertProviderType,
		list:    alertProviderListAdapter{&notificationv1.ProviderList{}},
	}.run,
}

func init() {
	getCmd.AddCommand(getAlertProviderCmd)
}

func (s alertProviderListAdapter) summariseItem(i int, includeNamespace bool, includeKind bool) []string {
	item := s.Items[i]
	status, msg := statusAndMessage(item.Status.Conditions)
	return append(nameColumns(&item, includeNamespace, includeKind), status, msg)
}

func (s alertProviderListAdapter) headers(includeNamespace bool) []string {
	headers := []string{"Name", "Ready", "Message"}
	if includeNamespace {
		return append(namespaceHeader, headers...)
	}
	return headers
}
