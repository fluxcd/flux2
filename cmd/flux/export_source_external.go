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
	"k8s.io/apimachinery/pkg/types"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var exportSourceExternalCmd = &cobra.Command{
	Use:   "external [name]",
	Short: "Export ExternalArtifact sources in YAML format",
	Long:  "The export source external command exports one or all ExternalArtifact sources in YAML format.",
	Example: `  # Export all ExternalArtifact sources
  flux export source external --all > sources.yaml

  # Export a specific ExternalArtifact
  flux export source external my-artifact > source.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.ExternalArtifactKind)),
	RunE: exportWithSecretCommand{
		list:   externalArtifactListAdapter{&sourcev1.ExternalArtifactList{}},
		object: externalArtifactAdapter{&sourcev1.ExternalArtifact{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceExternalCmd)
}

func exportExternalArtifact(source *sourcev1.ExternalArtifact) any {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.ExternalArtifactKind)
	export := sourcev1.ExternalArtifact{
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

func getExternalArtifactSecret(source *sourcev1.ExternalArtifact) *types.NamespacedName {
	// ExternalArtifact does not have a secretRef in its spec, this satisfies the interface
	return nil
}

func (ex externalArtifactAdapter) secret() *types.NamespacedName {
	return getExternalArtifactSecret(ex.ExternalArtifact)
}

func (ex externalArtifactListAdapter) secretItem(i int) *types.NamespacedName {
	return getExternalArtifactSecret(&ex.ExternalArtifactList.Items[i])
}

func (ex externalArtifactAdapter) export() any {
	return exportExternalArtifact(ex.ExternalArtifact)
}

func (ex externalArtifactListAdapter) exportItem(i int) any {
	return exportExternalArtifact(&ex.ExternalArtifactList.Items[i])
}
