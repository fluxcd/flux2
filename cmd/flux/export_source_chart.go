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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var exportSourceChartCmd = &cobra.Command{
	Use:   "chart [name]",
	Short: "Export HelmChart sources in YAML format",
	Long:  withPreviewNote("The export source chart command exports one or all HelmChart sources in YAML format."),
	Example: `  # Export all chart sources
  flux export source chart --all > sources.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind)),
	RunE: exportCommand{
		list:   helmChartListAdapter{&sourcev1.HelmChartList{}},
		object: helmChartAdapter{&sourcev1.HelmChart{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceChartCmd)
}

func exportHelmChart(source *sourcev1.HelmChart) interface{} {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.HelmChartKind)
	export := sourcev1.HelmChart{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        source.Name,
			Namespace:   source.Namespace,
			Labels:      source.Labels,
			Annotations: source.Annotations,
		},
		Spec: source.Spec,
	}
	return export
}

func (ex helmChartAdapter) export() interface{} {
	return exportHelmChart(ex.HelmChart)
}

func (ex helmChartListAdapter) exportItem(i int) interface{} {
	return exportHelmChart(&ex.HelmChartList.Items[i])
}
