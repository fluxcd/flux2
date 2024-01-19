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

var resumeAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Resume a suspended Provider",
	Long: `The resume command marks a previously suspended Provider resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Provider
  flux resume alert-provider main

  # Resume reconciliation for multiple Providers
  flux resume alert-provider main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: resumeCommand{
		apiType: alertProviderType,
		list:    &alertProviderListAdapter{&notificationv1.ProviderList{}},
	}.run,
}

func init() {
	resumeCmd.AddCommand(resumeAlertProviderCmd)
}

func (obj alertProviderAdapter) getObservedGeneration() int64 {
	return 0
}

func (obj alertProviderAdapter) setUnsuspended() {
	obj.Provider.Spec.Suspend = false
}

func (obj alertProviderAdapter) successMessage() string {
	return "Provider reconciliation completed"
}

func (a alertProviderAdapter) isStatic() bool {
	return true
}

func (a alertProviderListAdapter) resumeItem(i int) resumable {
	return &alertProviderAdapter{&a.ProviderList.Items[i]}
}
