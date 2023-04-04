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

var exportSourceBucketCmd = &cobra.Command{
	Use:   "bucket [name]",
	Short: "Export Bucket sources in YAML format",
	Long:  withPreviewNote("The export source git command exports one or all Bucket sources in YAML format."),
	Example: `  # Export all Bucket sources
  flux export source bucket --all > sources.yaml

  # Export a Bucket source including the static credentials
  flux export source bucket my-bucket --with-credentials > source.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)),
	RunE: exportWithSecretCommand{
		list:   bucketListAdapter{&sourcev1.BucketList{}},
		object: bucketAdapter{&sourcev1.Bucket{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceBucketCmd)
}

func exportBucket(source *sourcev1.Bucket) interface{} {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.BucketKind)
	export := sourcev1.Bucket{
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

func getBucketSecret(source *sourcev1.Bucket) *types.NamespacedName {
	if source.Spec.SecretRef != nil {
		namespacedName := types.NamespacedName{
			Namespace: source.Namespace,
			Name:      source.Spec.SecretRef.Name,
		}

		return &namespacedName
	}

	return nil
}

func (ex bucketAdapter) secret() *types.NamespacedName {
	return getBucketSecret(ex.Bucket)
}

func (ex bucketListAdapter) secretItem(i int) *types.NamespacedName {
	return getBucketSecret(&ex.BucketList.Items[i])
}

func (ex bucketAdapter) export() interface{} {
	return exportBucket(ex.Bucket)
}

func (ex bucketListAdapter) exportItem(i int) interface{} {
	return exportBucket(&ex.BucketList.Items[i])
}
