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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var exportSourceOCIRepositoryCmd = &cobra.Command{
	Use:   "oci [name]",
	Short: "Export OCIRepository sources in YAML format",
	Long:  withPreviewNote("The export source oci command exports one or all OCIRepository sources in YAML format."),
	Example: `  # Export all OCIRepository sources
  flux export source oci --all > sources.yaml

  # Export a OCIRepository including the static credentials
  flux export source oci my-app --with-credentials > source.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)),
	RunE: exportWithSecretCommand{
		list:   ociRepositoryListAdapter{&sourcev1.OCIRepositoryList{}},
		object: ociRepositoryAdapter{&sourcev1.OCIRepository{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceOCIRepositoryCmd)
}

func exportOCIRepository(source *sourcev1.OCIRepository) interface{} {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.OCIRepositoryKind)
	export := sourcev1.OCIRepository{
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

func getOCIRepositorySecret(source *sourcev1.OCIRepository) *types.NamespacedName {
	if source.Spec.SecretRef != nil {
		namespacedName := types.NamespacedName{
			Namespace: source.Namespace,
			Name:      source.Spec.SecretRef.Name,
		}

		return &namespacedName
	}

	return nil
}

func (ex ociRepositoryAdapter) secret() *types.NamespacedName {
	return getOCIRepositorySecret(ex.OCIRepository)
}

func (ex ociRepositoryListAdapter) secretItem(i int) *types.NamespacedName {
	return getOCIRepositorySecret(&ex.OCIRepositoryList.Items[i])
}

func (ex ociRepositoryAdapter) export() interface{} {
	return exportOCIRepository(ex.OCIRepository)
}

func (ex ociRepositoryListAdapter) exportItem(i int) interface{} {
	return exportOCIRepository(&ex.OCIRepositoryList.Items[i])
}
