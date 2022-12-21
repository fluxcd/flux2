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

var reconcileAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Reconcile an Alert",
	Long:  `The reconcile alert command triggers a reconciliation of an Alert resource and waits for it to finish.`,
	Example: `  # Trigger a reconciliation for an existing alert
  flux reconcile alert main`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.AlertKind)),
	RunE: reconcileCommand{
		apiType: alertType,
		object:  alertAdapter{&notificationv1.Alert{}},
	}.run,
}

func init() {
	reconcileCmd.AddCommand(reconcileAlertCmd)
}

func (obj alertAdapter) lastHandledReconcileRequest() string {
	return ""
}
