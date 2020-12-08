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

var resumeImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Resume a suspended ImageRepository",
	Long:  `The resume command marks a previously suspended ImageRepository resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing ImageRepository
  flux resume auto image-repository alpine
`,
	RunE: resumeCommand{
		kind:      imagev1.ImageRepositoryKind,
		humanKind: "image repository",
		object:    imageRepositoryAdapter{&imagev1.ImageRepository{}},
	}.run,
}

func init() {
	resumeAutoCmd.AddCommand(resumeImageRepositoryCmd)
}

func (obj imageRepositoryAdapter) getObservedGeneration() int64 {
	return obj.ImageRepository.Status.ObservedGeneration
}

func (obj imageRepositoryAdapter) setUnsuspended() {
	obj.ImageRepository.Spec.Suspend = false
}
