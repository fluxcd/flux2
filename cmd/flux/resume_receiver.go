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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1"
)

var resumeReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Resume a suspended Receiver",
	Long: `The resume command marks a previously suspended Receiver resource for reconciliation and waits for it to
finish the apply.`,
	Example: `  # Resume reconciliation for an existing Receiver
  flux resume receiver main

  # Resume reconciliation for multiple Receivers
  flux resume receiver main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind)),
	RunE: resumeCommand{
		apiType: receiverType,
		list:    receiverListAdapter{&notificationv1.ReceiverList{}},
	}.run,
}

func init() {
	resumeCmd.AddCommand(resumeReceiverCmd)
}

func (obj receiverAdapter) getObservedGeneration() int64 {
	return obj.Receiver.Status.ObservedGeneration
}

func (obj receiverAdapter) setUnsuspended() {
	obj.Receiver.Spec.Suspend = false
}

func (obj receiverAdapter) successMessage() string {
	return "Receiver reconciliation completed"
}

func (a receiverListAdapter) resumeItem(i int) resumable {
	return &receiverAdapter{&a.ReceiverList.Items[i]}
}
