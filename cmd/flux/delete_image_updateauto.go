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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1alpha1"
)

var deleteImageUpdateCmd = &cobra.Command{
	Use:   "image-update [name]",
	Short: "Delete an ImageUpdateAutomation object",
	Long:  "The delete auto image-update command deletes the given ImageUpdateAutomation from the cluster.",
	Example: `  # Delete an image update automation
  flux delete auto image-update latest-images
`,
	RunE: deleteCommand{
		names:  imageUpdateAutomationNames,
		object: universalAdapter{&autov1.ImageUpdateAutomation{}},
	}.run,
}

func init() {
	deleteAutoCmd.AddCommand(deleteImageUpdateCmd)
}
