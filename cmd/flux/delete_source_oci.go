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

var deleteSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Delete an OCIRepository source",
	Long:  withPreviewNote("The delete source oci command deletes the given OCIRepository from the cluster."),
	Example: `  # Delete an OCIRepository 
  flux delete source oci podinfo`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: deleteCommand{
		apiType: ociRepositoryType,
		object:  universalAdapter{&sourcev1.OCIRepository{}},
	}.run,
}

func init() {
	deleteSourceCmd.AddCommand(deleteSourceOCIRepositoryCmd)
}
