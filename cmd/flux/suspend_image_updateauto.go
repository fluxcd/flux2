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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
)

var suspendImageUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Suspend reconciliation of an ImageUpdateAutomation",
	Long:  `The suspend image update command disables the reconciliation of a ImageUpdateAutomation resource.`,
	Example: `  # Suspend reconciliation for an existing ImageUpdateAutomation
  flux suspend image update latest-images

  # Suspend reconciliation for multiple ImageUpdateAutomations
  flux suspend image update latest-images-1 latest-images-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)),
	RunE: suspendCommand{
		apiType: imageUpdateAutomationType,
		object:  imageUpdateAutomationAdapter{&autov1.ImageUpdateAutomation{}},
		list:    &imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{}},
	}.run,
}

func init() {
	suspendImageCmd.AddCommand(suspendImageUpdateCmd)
}

func (update imageUpdateAutomationAdapter) isSuspended() bool {
	return update.ImageUpdateAutomation.Spec.Suspend
}

func (update imageUpdateAutomationAdapter) setSuspended() {
	update.ImageUpdateAutomation.Spec.Suspend = true
}

func (a imageUpdateAutomationListAdapter) item(i int) suspendable {
	return &imageUpdateAutomationAdapter{&a.ImageUpdateAutomationList.Items[i]}
}
