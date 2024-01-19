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

var deleteSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Delete a HelmRepository source",
	Long:  withPreviewNote("The delete source helm command deletes the given HelmRepository from the cluster."),
	Example: `  # Delete a Helm repository
  flux delete source helm podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: deleteCommand{
		apiType: helmRepositoryType,
		object:  universalAdapter{&sourcev1.HelmRepository{}},
	}.run,
}

func init() {
	deleteSourceCmd.AddCommand(deleteSourceHelmCmd)
}
