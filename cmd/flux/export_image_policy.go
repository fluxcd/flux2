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

var exportImagePolicyCmd = &cobra.Command{
	Use:   "policy [name]",
	Short: "Export ImagePolicy resources in YAML format",
	Long:  withPreviewNote("The export image policy command exports one or all ImagePolicy resources in YAML format."),
	Example: `  # Export all ImagePolicy resources
  flux export image policy --all > image-policies.yaml

  # Export a specific policy
  flux export image policy alpine1x > alpine1x.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)),
	RunE: exportCommand{
		object: imagePolicyAdapter{&imagev1.ImagePolicy{}},
		list:   imagePolicyListAdapter{&imagev1.ImagePolicyList{}},
	}.run,
}

func init() {
	exportImageCmd.AddCommand(exportImagePolicyCmd)
}

// Export returns a ImagePolicy value which has extraneous information
// stripped out.
func exportImagePolicy(item *imagev1.ImagePolicy) interface{} {
	gvk := imagev1.GroupVersion.WithKind(imagev1.ImagePolicyKind)
	export := imagev1.ImagePolicy{
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

func (ex imagePolicyAdapter) export() interface{} {
	return exportImagePolicy(ex.ImagePolicy)
}

func (ex imagePolicyListAdapter) exportItem(i int) interface{} {
	return exportImagePolicy(&ex.ImagePolicyList.Items[i])
}
