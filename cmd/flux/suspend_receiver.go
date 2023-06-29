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

var suspendReceiverCmd = &cobra.Command{
	Use:   "receiver [name]",
	Short: "Suspend reconciliation of Receiver",
	Long:  `The suspend command disables the reconciliation of a Receiver resource.`,
	Example: `  # Suspend reconciliation for an existing Receiver
  flux suspend receiver main

  # Suspend reconciliation for multiple Receivers
  flux suspend receiver main-1 main-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(notificationv1.GroupVersion.WithKind(notificationv1.ReceiverKind)),
	RunE: suspendCommand{
		apiType: receiverType,
		object:  &receiverAdapter{&notificationv1.Receiver{}},
		list:    &receiverListAdapter{&notificationv1.ReceiverList{}},
	}.run,
}

func init() {
	suspendCmd.AddCommand(suspendReceiverCmd)
}

func (obj receiverAdapter) isSuspended() bool {
	return obj.Receiver.Spec.Suspend
}

func (obj receiverAdapter) setSuspended() {
	obj.Receiver.Spec.Suspend = true
}

func (a receiverListAdapter) item(i int) suspendable {
	return &receiverAdapter{&a.ReceiverList.Items[i]}
}
