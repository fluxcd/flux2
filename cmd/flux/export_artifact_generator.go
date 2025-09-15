/*
Copyright 2025 The Flux authors

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

	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

var exportArtifactGeneratorCmd = &cobra.Command{
	Use:   "generator [name]",
	Short: "Export ArtifactGenerator resources in YAML format",
	Long:  "The export artifact generator command exports one or all ArtifactGenerator resources in YAML format.",
	Example: `  # Export all ArtifactGenerator resources
  flux export artifact generator --all > artifact-generators.yaml

  # Export a specific generator
  flux export artifact generator my-generator > my-generator.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)),
	RunE: exportCommand{
		object: artifactGeneratorAdapter{&swapi.ArtifactGenerator{}},
		list:   artifactGeneratorListAdapter{&swapi.ArtifactGeneratorList{}},
	}.run,
}

func init() {
	exportArtifactCmd.AddCommand(exportArtifactGeneratorCmd)
}

// Export returns an ArtifactGenerator value which has
// extraneous information stripped out.
func exportArtifactGenerator(item *swapi.ArtifactGenerator) interface{} {
	gvk := swapi.GroupVersion.WithKind(swapi.ArtifactGeneratorKind)
	export := swapi.ArtifactGenerator{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        item.Name,
			Namespace:   item.Namespace,
			Labels:      item.Labels,
			Annotations: item.Annotations,
		},
		Spec: item.Spec,
	}
	return export
}

func (ex artifactGeneratorAdapter) export() interface{} {
	return exportArtifactGenerator(ex.ArtifactGenerator)
}

func (ex artifactGeneratorListAdapter) exportItem(i int) interface{} {
	return exportArtifactGenerator(&ex.ArtifactGeneratorList.Items[i])
}
