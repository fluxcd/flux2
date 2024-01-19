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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta3"
)

var resumeAlertCmd = &cobra.Command{
	Use:   "alert [name]",
	Short: "Resume a suspended Alert",
	Long: `The resume command marks a previously suspended Alert resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Alert
  flux resume alert main

  # Resume reconciliation for multiple Alerts
  flux resume alert main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.AlertKind)),
	RunE: resumeCommand{
		apiType: alertType,
		list:    &alertListAdapter{&notificationv1.AlertList{}},
	}.run,
}

func init() {
	resumeCmd.AddCommand(resumeAlertCmd)
}

func (obj alertAdapter) getObservedGeneration() int64 {
	return 0
}

func (obj alertAdapter) setUnsuspended() {
	obj.Alert.Spec.Suspend = false
}

func (obj alertAdapter) successMessage() string {
	return "Alert reconciliation completed"
}

func (a alertAdapter) isStatic() bool {
	return true
}

func (a alertListAdapter) resumeItem(i int) resumable {
	return &alertAdapter{&a.AlertList.Items[i]}
}
