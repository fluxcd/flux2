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

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the toolkit components",
	Long:  "The uninstall command removes the namespace, cluster roles, cluster role bindings and CRDs from the cluster.",
	Example: `  # Dry-run uninstall of all components
  gotk uninstall --dry-run --namespace=gotk-system

  # Uninstall all components and delete custom resource definitions
  gotk uninstall --resources --crds --namespace=gotk-system
`,
	RunE: uninstallCmdRun,
}

var (
	uninstallCRDs      bool
	uninstallResources bool
	uninstallDryRun    bool
	uninstallSilent    bool
)

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallResources, "resources", true,
		"removes custom resources such as Kustomizations, GitRepositories and HelmRepositories")
	uninstallCmd.Flags().BoolVar(&uninstallCRDs, "crds", false,
		"removes all CRDs previously installed")
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false,
		"only print the object that would be deleted")
	uninstallCmd.Flags().BoolVarP(&uninstallSilent, "silent", "s", false,
		"delete components without asking for confirmation")

	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kubeClient, err := utils.kubeClient(kubeconfig)
	if err != nil {
		return err
	}

	dryRun := ""
	if uninstallDryRun {
		dryRun = "--dry-run=server"
	} else if !uninstallSilent {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to delete the %s namespace", namespace),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	// suspend bootstrap kustomization if it exists
	kustomizationName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	var kustomization kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, kustomizationName, &kustomization); err == nil {
		kustomization.Spec.Suspend = true
		if err := kubeClient.Update(ctx, &kustomization); err != nil {
			return fmt.Errorf("unable to suspend kustomization '%s': %w", kustomizationName.String(), err)
		}
	}

	if uninstallResources || uninstallCRDs {
		logger.Actionf("uninstalling custom resources")
		for _, kind := range []string{
			kustomizev1.KustomizationKind,
			sourcev1.GitRepositoryKind,
			sourcev1.HelmRepositoryKind,
			helmv2.HelmReleaseKind,
		} {
			command := fmt.Sprintf("kubectl -n %s delete %s --all --ignore-not-found --timeout=%s %s",
				namespace, kind, timeout.String(), dryRun)
			if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
				return fmt.Errorf("uninstall failed: %w", err)
			}
		}
	}

	var kinds []string
	if uninstallCRDs {
		kinds = append(kinds, "crds")
	}

	kinds = append(kinds, "clusterroles,clusterrolebindings", "namespace")

	logger.Actionf("uninstalling components")

	for _, kind := range kinds {
		command := fmt.Sprintf("kubectl delete %s -l app.kubernetes.io/instance=%s --ignore-not-found --timeout=%s %s",
			kind, namespace, timeout.String(), dryRun)
		if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}
	}

	logger.Successf("uninstall finished")
	return nil
}
