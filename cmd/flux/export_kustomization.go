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

var exportKsCmd = &cobra.Command{
	Use:     "kustomization [name]",
	Aliases: []string{"ks"},
	Short:   "Export Kustomization resources in YAML format",
	Long:    `The export kustomization command exports one or all Kustomization resources in YAML format.`,
	Example: `  # Export all Kustomization resources
  flux export kustomization --all > kustomizations.yaml

  # Export a Kustomization
  flux export kustomization my-app > kustomization.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE: exportCommand{
		object: kustomizationAdapter{&kustomizev1.Kustomization{}},
		list:   kustomizationListAdapter{&kustomizev1.KustomizationList{}},
	}.run,
}

func init() {
	exportCmd.AddCommand(exportKsCmd)
}

func exportKs(kustomization *kustomizev1.Kustomization) any {
	gvk := kustomizev1.GroupVersion.WithKind("Kustomization")
	export := kustomizev1.Kustomization{
		Kind:        gvk.Kind,
		APIVersion:  gvk.GroupVersion().String(),
		Name:        kustomization.Name,
		Namespace:   kustomization.Namespace,
		Labels:      kustomization.Labels,
		Annotations: kustomization.Annotations,
		Spec:        kustomization.Spec,
	}

	return export
}

func (ex kustomizationAdapter) export() any {
	return exportKs(ex.Kustomization)
}

func (ex kustomizationListAdapter) exportItem(i int) any {
	return exportKs(&ex.KustomizationList.Items[i])
}
