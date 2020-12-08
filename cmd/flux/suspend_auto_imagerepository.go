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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var suspendImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Suspend reconciliation of an ImageRepository",
	Long:  "The suspend command disables the reconciliation of a ImageRepository resource.",
	Example: `  # Suspend reconciliation for an existing ImageRepository
  flux suspend auto image-repository alpine
`,
	RunE: suspendCommand{
		names:  imageRepositoryNames,
		object: imageRepositoryAdapter{&imagev1.ImageRepository{}},
	}.run,
}

func init() {
	suspendAutoCmd.AddCommand(suspendImageRepositoryCmd)
}

func (obj imageRepositoryAdapter) isSuspended() bool {
	return obj.ImageRepository.Spec.Suspend
}

func (obj imageRepositoryAdapter) setSuspended() {
	obj.ImageRepository.Spec.Suspend = true
}
