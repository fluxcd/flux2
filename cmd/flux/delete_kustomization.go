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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
)

var deleteKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Delete a Kustomization resource",
	Long:    `The delete kustomization command deletes the given Kustomization from the cluster.`,
	Example: `  # Delete a kustomization and the Kubernetes resources created by it when prune is enabled
  flux delete kustomization podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE: deleteCommand{
		apiType: kustomizationType,
		object:  universalAdapter{&kustomizev1.Kustomization{}},
	}.run,
}

func init() {
	deleteCmd.AddCommand(deleteKsCmd)
}
