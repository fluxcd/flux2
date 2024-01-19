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

var deleteSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Delete a GitRepository source",
	Long:  `The delete source git command deletes the given GitRepository from the cluster.`,
	Example: `  # Delete a Git repository
  flux delete source git podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)),
	RunE: deleteCommand{
		apiType: gitRepositoryType,
		object:  universalAdapter{&sourcev1.GitRepository{}},
	}.run,
}

func init() {
	deleteSourceCmd.AddCommand(deleteSourceGitCmd)
}
