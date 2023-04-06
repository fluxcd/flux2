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

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta2"
)

var exportImageRepositoryCmd = &cobra.Command{
	Use:   "repository [name]",
	Short: "Export ImageRepository resources in YAML format",
	Long:  withPreviewNote("The export image repository command exports one or all ImageRepository resources in YAML format."),
	Example: `  # Export all ImageRepository resources
  flux export image repository --all > image-repositories.yaml

  # Export a specific ImageRepository resource
  flux export image repository alpine > alpine.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind)),
	RunE: exportCommand{
		object: imageRepositoryAdapter{&imagev1.ImageRepository{}},
		list:   imageRepositoryListAdapter{&imagev1.ImageRepositoryList{}},
	}.run,
}

func init() {
	exportImageCmd.AddCommand(exportImageRepositoryCmd)
}

func exportImageRepository(repo *imagev1.ImageRepository) interface{} {
	gvk := imagev1.GroupVersion.WithKind(imagev1.ImageRepositoryKind)
	export := imagev1.ImageRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        repo.Name,
			Namespace:   repo.Namespace,
			Labels:      repo.Labels,
			Annotations: repo.Annotations,
		},
		Spec: repo.Spec,
	}
	return export
}

func (ex imageRepositoryAdapter) export() interface{} {
	return exportImageRepository(ex.ImageRepository)
}

func (ex imageRepositoryListAdapter) exportItem(i int) interface{} {
	return exportImageRepository(&ex.ImageRepositoryList.Items[i])
}
