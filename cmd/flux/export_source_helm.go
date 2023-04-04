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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
)

var exportSourceHelmCmd = &cobra.Command{
	Use:   "helm [name]",
	Short: "Export HelmRepository sources in YAML format",
	Long:  withPreviewNote("The export source git command exports one or all HelmRepository sources in YAML format."),
	Example: `  # Export all HelmRepository sources
  flux export source helm --all > sources.yaml

  # Export a HelmRepository source including the basic auth credentials
  flux export source helm my-private-repo --with-credentials > source.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)),
	RunE: exportWithSecretCommand{
		list:   helmRepositoryListAdapter{&sourcev1.HelmRepositoryList{}},
		object: helmRepositoryAdapter{&sourcev1.HelmRepository{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceHelmCmd)
}

func exportHelmRepository(source *sourcev1.HelmRepository) interface{} {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.HelmRepositoryKind)
	export := sourcev1.HelmRepository{
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

func getHelmSecret(source *sourcev1.HelmRepository) *types.NamespacedName {
	if source.Spec.SecretRef != nil {
		namespacedName := types.NamespacedName{
			Namespace: source.Namespace,
			Name:      source.Spec.SecretRef.Name,
		}
		return &namespacedName
	}
	return nil
}

func (ex helmRepositoryAdapter) secret() *types.NamespacedName {
	return getHelmSecret(ex.HelmRepository)
}

func (ex helmRepositoryListAdapter) secretItem(i int) *types.NamespacedName {
	return getHelmSecret(&ex.HelmRepositoryList.Items[i])
}

func (ex helmRepositoryAdapter) export() interface{} {
	return exportHelmRepository(ex.HelmRepository)
}

func (ex helmRepositoryListAdapter) exportItem(i int) interface{} {
	return exportHelmRepository(&ex.HelmRepositoryList.Items[i])
}
