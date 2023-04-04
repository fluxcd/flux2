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

var deleteImageUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Delete an ImageUpdateAutomation object",
	Long:  withPreviewNote(`The delete image update command deletes the given ImageUpdateAutomation from the cluster.`),
	Example: `  # Delete an image update automation
  flux delete image update latest-images`,
	ValidArgsFunction: resourceNamesCompletionFunc(autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)),
	RunE: deleteCommand{
		apiType: imageUpdateAutomationType,
		object:  universalAdapter{&autov1.ImageUpdateAutomation{}},
	}.run,
}

func init() {
	deleteImageCmd.AddCommand(deleteImageUpdateCmd)
}
