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

var resumeSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Resume a suspended HelmRepository",
	Long:  `The resume command marks a previously suspended HelmRepository resource for reconciliation and waits for it to finish.`,
	Example: `  # Resume reconciliation for an existing HelmRepository
  flux resume source helm bitnami

  # Resume reconciliation for multiple HelmRepositories
  flux resume source helm bitnami-1 bitnami-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: resumeCommand{
		apiType: helmRepositoryType,
		list:    helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{}},
	}.run,
}

func init() {
	resumeSourceCmd.AddCommand(resumeSourceHelmCmd)
}

func (obj helmRepositoryAdapter) getObservedGeneration() int64 {
	return obj.HelmRepository.Status.ObservedGeneration
}

func (obj helmRepositoryAdapter) setUnsuspended() {
	obj.HelmRepository.Spec.Suspend = false
}

func (a helmRepositoryListAdapter) resumeItem(i int) resumable {
	return &helmRepositoryAdapter{&a.HelmRepositoryList.Items[i]}
}
