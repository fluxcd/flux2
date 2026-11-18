/*
Copyright 2024 The Flux authors

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

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
)

var exportHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Export HelmRelease resources in YAML format",
	Long:    "The export helmrelease command exports one or all HelmRelease resources in YAML format.",
	Example: `  # Export all HelmRelease resources
  flux export helmrelease --all > kustomizations.yaml

  # Export a HelmRelease
  flux export hr my-app > app-release.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)),
	RunE: exportCommand{
		object: helmReleaseAdapter{&helmv2.HelmRelease{}},
		list:   helmReleaseListAdapter{&helmv2.HelmReleaseList{}},
	}.run,
}

func init() {
	exportCmd.AddCommand(exportHelmReleaseCmd)
}

func exportHelmRelease(helmRelease *helmv2.HelmRelease) any {
	gvk := helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)
	export := helmv2.HelmRelease{
		Kind:        gvk.Kind,
		APIVersion:  gvk.GroupVersion().String(),
		Name:        helmRelease.Name,
		Namespace:   helmRelease.Namespace,
		Labels:      helmRelease.Labels,
		Annotations: helmRelease.Annotations,
		Spec:        helmRelease.Spec,
	}
	return export
}

func (ex helmReleaseAdapter) export() any {
	return exportHelmRelease(ex.HelmRelease)
}

func (ex helmReleaseListAdapter) exportItem(i int) any {
	return exportHelmRelease(&ex.HelmReleaseList.Items[i])
}
