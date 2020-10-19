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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/fluxcd/toolkit/internal/utils"
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

	kubeClient, err := utils.KubeClient(kubeconfig)
	if err != nil {
		return err
	}

	if !uninstallDryRun && !uninstallSilent {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to delete the %s namespace", namespace),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	dryRun := "--dry-run=server"
	deleteResources := uninstallResources || uninstallCRDs

	// known kinds with finalizers
	namespacedKinds := []string{
		sourcev1.GitRepositoryKind,
		sourcev1.HelmRepositoryKind,
		sourcev1.BucketKind,
	}

	// suspend bootstrap kustomization to avoid finalizers deadlock
	kustomizationName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	var kustomization kustomizev1.Kustomization
	err = kubeClient.Get(ctx, kustomizationName, &kustomization)
	if err == nil {
		kustomization.Spec.Suspend = true
		if err := kubeClient.Update(ctx, &kustomization); err != nil {
			return fmt.Errorf("unable to suspend kustomization '%s': %w", kustomizationName.String(), err)
		}
	}
	if err == nil || apierrors.IsNotFound(err) {
		namespacedKinds = append(namespacedKinds, kustomizev1.KustomizationKind)
	}

	// add HelmRelease kind to deletion list if exists
	var list helmv2.HelmReleaseList
	if err := kubeClient.List(ctx, &list, client.InNamespace(namespace)); err == nil {
		namespacedKinds = append(namespacedKinds, helmv2.HelmReleaseKind)
	}

	if deleteResources {
		logger.Actionf("uninstalling custom resources")
		for _, kind := range namespacedKinds {
			if err := deleteAll(ctx, kind, uninstallDryRun); err != nil {
				logger.Failuref("kubectl: %s", err.Error())
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
		kubectlArgs := []string{
			"delete", kind,
			"-l", fmt.Sprintf("app.kubernetes.io/instance=%s", namespace),
			"--ignore-not-found", "--timeout", timeout.String(),
		}
		if uninstallDryRun {
			kubectlArgs = append(kubectlArgs, dryRun)
		}
		if _, err := utils.ExecKubectlCommand(ctx, utils.ModeOS, kubectlArgs...); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}
	}

	logger.Successf("uninstall finished")
	return nil
}

func deleteAll(ctx context.Context, kind string, dryRun bool) error {
	kubectlArgs := []string{
		"delete", kind, "--ignore-not-found",
		"--all", "--all-namespaces",
		"--timeout", timeout.String(),
	}

	if dryRun {
		kubectlArgs = append(kubectlArgs, "--dry-run=server")
	}

	_, err := utils.ExecKubectlCommand(ctx, utils.ModeOS, kubectlArgs...)
	return err
}
