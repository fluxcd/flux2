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

var suspendSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Suspend reconciliation of an OCIRepository",
	Long:  `The suspend command disables the reconciliation of an OCIRepository resource.`,
	Example: `  # Suspend reconciliation for an existing OCIRepository
  flux suspend source oci podinfo

  # Suspend reconciliation for multiple OCIRepositories
  flux suspend source oci podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: suspendCommand{
		apiType: ociRepositoryType,
		object:  ociRepositoryAdapter{&sourcev1.OCIRepository{}},
		list:    ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{}},
	}.run,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceOCIRepositoryCmd)
}

func (obj ociRepositoryAdapter) isSuspended() bool {
	return obj.OCIRepository.Spec.Suspend
}

func (obj ociRepositoryAdapter) setSuspended() {
	obj.OCIRepository.Spec.Suspend = true
}

func (a ociRepositoryListAdapter) item(i int) suspendable {
	return &ociRepositoryAdapter{&a.OCIRepositoryList.Items[i]}
}
