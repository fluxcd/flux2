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

var resumeImageUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Resume a suspended ImageUpdateAutomation",
	Long:  `The resume command marks a previously suspended ImageUpdateAutomation resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing ImageUpdateAutomation
  flux resume image update latest-images

  # Resume reconciliation for multiple ImageUpdateAutomations
  flux resume image update latest-images-1 latest-images-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)),
	RunE: resumeCommand{
		apiType: imageUpdateAutomationType,
		list:    imageUpdateAutomationListAdapter{&autov1.ImageUpdateAutomationList{}},
	}.run,
}

func init() {
	resumeImageCmd.AddCommand(resumeImageUpdateCmd)
}

func (obj imageUpdateAutomationAdapter) setUnsuspended() {
	obj.ImageUpdateAutomation.Spec.Suspend = false
}

func (obj imageUpdateAutomationAdapter) getObservedGeneration() int64 {
	return obj.ImageUpdateAutomation.Status.ObservedGeneration
}

func (a imageUpdateAutomationListAdapter) resumeItem(i int) resumable {
	return &imageUpdateAutomationAdapter{&a.ImageUpdateAutomationList.Items[i]}
}
