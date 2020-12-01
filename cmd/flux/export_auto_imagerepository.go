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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/internal/utils"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
)

var exportImageRepositoryCmd = &cobra.Command{
	Use:   "image-repository [name]",
	Short: "Export ImageRepository resources in YAML format",
	Long:  "The export image-repository command exports one or all ImageRepository resources in YAML format.",
	Example: `  # Export all ImageRepository resources
  flux export auto image-repository --all > image-repositories.yaml

  # Export a Provider
  flux export auto image-repository alpine > alpine.yaml
`,
	RunE: exportImageRepositoryCmdRun,
}

func init() {
	exportAutoCmd.AddCommand(exportImageRepositoryCmd)
}

func exportImageRepositoryCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return err
	}

	if exportAll {
		var list imagev1.ImageRepositoryList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logger.Failuref("no imagerepository objects found in %s namespace", namespace)
			return nil
		}

		for _, imageRepo := range list.Items {
			if err := exportImageRepo(imageRepo); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var imageRepo imagev1.ImageRepository
		err = kubeClient.Get(ctx, namespacedName, &imageRepo)
		if err != nil {
			return err
		}
		return exportImageRepo(imageRepo)
	}
	return nil
}

func exportImageRepo(repo imagev1.ImageRepository) error {
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

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
