/*
Copyright 2022 The Flux authors

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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var resumeSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Resume a suspended OCIRepository",
	Long:  `The resume command marks a previously suspended OCIRepository resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing OCIRepository
  flux resume source oci podinfo

  # Resume reconciliation for multiple OCIRepositories
  flux resume source oci podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: resumeCommand{
		apiType: ociRepositoryType,
		list:    ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{}},
	}.run,
}

func init() {
	resumeSourceCmd.AddCommand(resumeSourceOCIRepositoryCmd)
}

func (obj ociRepositoryAdapter) getObservedGeneration() int64 {
	return obj.OCIRepository.Status.ObservedGeneration
}

func (obj ociRepositoryAdapter) setUnsuspended() {
	obj.OCIRepository.Spec.Suspend = false
}

func (a ociRepositoryListAdapter) resumeItem(i int) resumable {
	return &ociRepositoryAdapter{&a.OCIRepositoryList.Items[i]}
}
