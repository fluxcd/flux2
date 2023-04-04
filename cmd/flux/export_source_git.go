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

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

var exportSourceGitCmd = &cobra.Command{
	Use:   "git [name]",
	Short: "Export GitRepository sources in YAML format",
	Long:  `The export source git command exports one or all GitRepository sources in YAML format.`,
	Example: `  # Export all GitRepository sources
  flux export source git --all > sources.yaml

  # Export a GitRepository source including the SSH key pair or basic auth credentials
  flux export source git my-private-repo --with-credentials > source.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)),
	RunE: exportWithSecretCommand{
		object: gitRepositoryAdapter{&sourcev1.GitRepository{}},
		list:   gitRepositoryListAdapter{&sourcev1.GitRepositoryList{}},
	}.run,
}

func init() {
	exportSourceCmd.AddCommand(exportSourceGitCmd)
}

func exportGit(source *sourcev1.GitRepository) interface{} {
	gvk := sourcev1.GroupVersion.WithKind(sourcev1.GitRepositoryKind)
	export := sourcev1.GitRepository{
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

func getGitSecret(source *sourcev1.GitRepository) *types.NamespacedName {
	if source.Spec.SecretRef != nil {
		namespacedName := types.NamespacedName{
			Namespace: source.Namespace,
			Name:      source.Spec.SecretRef.Name,
		}
		return &namespacedName
	}

	return nil
}

func (ex gitRepositoryAdapter) secret() *types.NamespacedName {
	return getGitSecret(ex.GitRepository)
}

func (ex gitRepositoryListAdapter) secretItem(i int) *types.NamespacedName {
	return getGitSecret(&ex.GitRepositoryList.Items[i])
}

func (ex gitRepositoryAdapter) export() interface{} {
	return exportGit(ex.GitRepository)
}

func (ex gitRepositoryListAdapter) exportItem(i int) interface{} {
	return exportGit(&ex.GitRepositoryList.Items[i])
}
