/*
Copyright 2023 The Flux authors

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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var suspendAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Suspend reconciliation of Provider",
	Long:  `The suspend command disables the reconciliation of a Provider resource.`,
	Example: `  # Suspend reconciliation for an existing Provider
  flux suspend alert-provider main

  # Suspend reconciliation for multiple Providers
  flux suspend alert-providers main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: suspendCommand{
		apiType: alertProviderType,
		object:  &alertProviderAdapter{&notificationv1.Provider{}},
		list:    &alertProviderListAdapter{&notificationv1.ProviderList{}},
	}.run,
}

func init() {
	suspendCmd.AddCommand(suspendAlertProviderCmd)
}

func (obj alertProviderAdapter) isSuspended() bool {
	return obj.Provider.Spec.Suspend
}

func (obj alertProviderAdapter) setSuspended() {
	obj.Provider.Spec.Suspend = true
}

func (a alertProviderListAdapter) item(i int) suspendable {
	return &alertProviderAdapter{&a.ProviderList.Items[i]}
}
