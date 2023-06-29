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

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var suspendSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Suspend reconciliation of a GitRepository",
	Long:  `The suspend command disables the reconciliation of a GitRepository resource.`,
	Example: `  # Suspend reconciliation for an existing GitRepository
  flux suspend source git podinfo

  # Suspend reconciliation for multiple GitRepositories
  flux suspend source git podinfo-1 podinfo-2`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)),
	RunE: suspendCommand{
		apiType: gitRepositoryType,
		object:  gitRepositoryAdapter{&sourcev1.GitRepository{}},
		list:    gitRepositoryListAdapter{&sourcev1.GitRepositoryList{}},
	}.run,
}

func init() {
	suspendSourceCmd.AddCommand(suspendSourceGitCmd)
}

func (obj gitRepositoryAdapter) isSuspended() bool {
	return obj.GitRepository.Spec.Suspend
}

func (obj gitRepositoryAdapter) setSuspended() {
	obj.GitRepository.Spec.Suspend = true
}

func (a gitRepositoryListAdapter) item(i int) suspendable {
	return &gitRepositoryAdapter{&a.GitRepositoryList.Items[i]}
}
