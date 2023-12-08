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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta2"
)

var reconcileAlertProviderCmd = &cobra.Command{
	Use:   "alert-provider [name]",
	Short: "Reconcile a Provider",
	Long:  `The reconcile alert-provider command triggers a reconciliation of a Provider resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing provider
  flux reconcile alert-provider slack`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ProviderKind)),
	RunE: reconcileCommand{
		apiType: alertProviderType,
		object:  alertProviderAdapter{&notificationv1.Provider{}},
	}.run,
}

func init() {
	reconcileCmd.AddCommand(reconcileAlertProviderCmd)
}

func (obj alertProviderAdapter) lastHandledReconcileRequest() string {
	return obj.Status.GetLastHandledReconcileRequest()
}
