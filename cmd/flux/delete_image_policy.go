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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var deleteImagePolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Delete an ImagePolicy object",
	Long:  withPreviewNote(`The delete image policy command deletes the given ImagePolicy from the cluster.`),
	Example: `  # Delete an image policy
  flux delete image policy alpine3.x`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: deleteCommand{
		apiType: imagePolicyType,
		object:  universalAdapter{&imagev1.ImagePolicy{}},
	}.run,
}

func init() {
	deleteImageCmd.AddCommand(deleteImagePolicyCmd)
}
