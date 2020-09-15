/*
Copyright 2020 The Flux CD contributors.

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

	helmv2 "github.com/fluxcd/helm-controller/api/v2alpha1"
)

var exportHelmReleaseCmd = &cobra.Command{
	Use:     "helmrelease [name]",
	Aliases: []string{"hr"},
	Short:   "Export HelmRelease resources in YAML format",
	Long:    "The export helmrelease command exports one or all HelmRelease resources in YAML format.",
	Example: `  # Export all HelmRelease resources
  gotk export helmrelease --all > kustomizations.yaml

  # Export a HelmRelease
  gotk export hr my-app > app-release.yaml
`,
	RunE: exportHelmReleaseCmdRun,
}

func init() {
	exportCmd.AddCommand(exportHelmReleaseCmd)
}

func exportHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	if !exportAll && len(args) < 1 {
		return fmt.Errorf("name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if exportAll {
		var list helmv2.HelmReleaseList
		err = kubeClient.List(ctx, &list, client.InNamespace(namespace))
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			logger.Failuref("no helmrelease found in %s namespace", namespace)
			return nil
		}

		for _, helmRelease := range list.Items {
			if err := exportHelmRelease(helmRelease); err != nil {
				return err
			}
		}
	} else {
		name := args[0]
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		var helmRelease helmv2.HelmRelease
		err = kubeClient.Get(ctx, namespacedName, &helmRelease)
		if err != nil {
			return err
		}
		return exportHelmRelease(helmRelease)
	}
	return nil
}

func exportHelmRelease(helmRelease helmv2.HelmRelease) error {
	gvk := helmv2.GroupVersion.WithKind(helmv2.HelmReleaseKind)
	export := helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        helmRelease.Name,
			Namespace:   helmRelease.Namespace,
			Labels:      helmRelease.Labels,
			Annotations: helmRelease.Annotations,
		},
		Spec: helmRelease.Spec,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return err
	}

	fmt.Println("---")
	fmt.Println(resourceToString(data))
	return nil
}
