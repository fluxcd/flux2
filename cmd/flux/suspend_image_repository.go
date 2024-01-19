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

var suspendImageRepositoryCmd = &cobra.Command{
	Use:   "repository [name]",
	Short: "Suspend reconciliation of an ImageRepository",
	Long:  `The suspend image repository command disables the reconciliation of a ImageRepository resource.`,
	Example: `  # Suspend reconciliation for an existing ImageRepository
  flux suspend image repository alpine

  # Suspend reconciliation for multiple ImageRepositories
  flux suspend image repository alpine-1 alpine-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind)),
	RunE: suspendCommand{
		apiType: imageRepositoryType,
		object:  imageRepositoryAdapter{&imagev1.ImageRepository{}},
		list:    &imageRepositoryListAdapter{&imagev1.ImageRepositoryList{}},
	}.run,
}

func init() {
	suspendImageCmd.AddCommand(suspendImageRepositoryCmd)
}

func (obj imageRepositoryAdapter) isSuspended() bool {
	return obj.ImageRepository.Spec.Suspend
}

func (obj imageRepositoryAdapter) setSuspended() {
	obj.ImageRepository.Spec.Suspend = true
}

func (a imageRepositoryListAdapter) item(i int) suspendable {
	return &imageRepositoryAdapter{&a.ImageRepositoryList.Items[i]}
}
