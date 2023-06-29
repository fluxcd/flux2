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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var suspendSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Suspend reconciliation of a HelmRepository",
	Long:  `The suspend command disables the reconciliation of a HelmRepository resource.`,
	Example: `  # Suspend reconciliation for an existing HelmRepository
  flux suspend source helm bitnami

  # Suspend reconciliation for multiple HelmRepositories
  flux suspend source helm bitnami-1 bitnami-2 `,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: suspendCommand{
		apiType: helmRepositoryType,
		object:  helmRepositoryAdapter{&sourcev1.HelmRepository{}},
		list:    helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{}},
	}.run,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceHelmCmd)
}

func (obj helmRepositoryAdapter) isSuspended() bool {
	return obj.HelmRepository.Spec.Suspend
}

func (obj helmRepositoryAdapter) setSuspended() {
	obj.HelmRepository.Spec.Suspend = true
}

func (a helmRepositoryListAdapter) item(i int) suspendable {
	return &helmRepositoryAdapter{&a.HelmRepositoryList.Items[i]}
}
