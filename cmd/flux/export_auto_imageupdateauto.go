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
	"k8s.io/apimachinery/pkg/runtime"

	autov1 "github.com/fluxcd/image-automation-controller/api/v1alpha1"
)

var exportImageUpdateCmd = &cobra.Command{
	Use:   "image-update [name]",
	Short: "Export ImageUpdateAutomation resources in YAML format",
	Long:  "The export image-update command exports one or all ImageUpdateAutomation resources in YAML format.",
	Example: `  # Export all ImageUpdateAutomation resources
  flux export auto image-update --all > updates.yaml

  # Export a specific automation
  flux export auto image-update latest-images > latest.yaml
`,
	RunE: exportCommand{
		object: exportableImageUpdate{&autov1.ImageUpdateAutomation{}},
		list:   exportableImageUpdateList{&autov1.ImageUpdateAutomationList{}},
	}.run,
}

func init() {
	exportAutoCmd.AddCommand(exportImageUpdateCmd)
}

// exportImageUpdate returns a value which has extraneous information
// stripped out.
func exportImageUpdate(item *autov1.ImageUpdateAutomation) interface{} {
	gvk := autov1.GroupVersion.WithKind(autov1.ImageUpdateAutomationKind)
	export := autov1.ImageUpdateAutomation{
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

type exportableImageUpdate struct {
	update *autov1.ImageUpdateAutomation
}

func (ex exportableImageUpdate) AsClientObject() runtime.Object {
	return ex.update
}

func (ex exportableImageUpdate) Export() interface{} {
	return exportImageUpdate(ex.update)
}

type exportableImageUpdateList struct {
	list *autov1.ImageUpdateAutomationList
}

func (ex exportableImageUpdateList) AsClientObject() runtime.Object {
	return ex.list
}

func (ex exportableImageUpdateList) Len() int {
	return len(ex.list.Items)
}

func (ex exportableImageUpdateList) ExportAt(i int) interface{} {
	return exportImageUpdate(&ex.list.Items[i])
}
