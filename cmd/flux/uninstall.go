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

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/fluxcd/flux2/v2/internal/utils"
	"github.com/fluxcd/flux2/v2/pkg/uninstall"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Args:  cobra.NoArgs,
	Short: "Uninstall Flux and its custom resource definitions",
	Long:  `The uninstall command removes the Flux components and the toolkit.fluxcd.io resources from the cluster.`,
	Example: `  # Uninstall Flux components, its custom resources and namespace
  flux uninstall --namespace=flux-system

  # Uninstall Flux but keep the namespace
  flux uninstall --namespace=infra --keep-namespace=true`,
	RunE: uninstallCmdRun,
}

type uninstallFlags struct {
	keepNamespace bool
	dryRun        bool
	silent        bool
}

var uninstallArgs uninstallFlags

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallArgs.keepNamespace, "keep-namespace", false,
		"skip namespace deletion")
	uninstallCmd.Flags().BoolVar(&uninstallArgs.dryRun, "dry-run", false,
		"only print the objects that would be deleted")
	uninstallCmd.Flags().BoolVarP(&uninstallArgs.silent, "silent", "s", false,
		"delete components without asking for confirmation")

	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(kubeconfigArgs, kubeclientOptions)
	if err != nil {
		return err
	}

	if !uninstallArgs.dryRun && !uninstallArgs.silent {
		info, err := getFluxClusterInfo(ctx, kubeClient)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("cluster info unavailable: %w", err)
			}
		}

		promptLabel := "Are you sure you want to delete Flux and its custom resource definitions"
		if !installManagedByFlux(info.managedBy) {
			promptLabel = fmt.Sprintf("Flux is managed by %s! Are you sure you want to delete Flux and its CRDs using Flux CLI", info.managedBy)
		}
		prompt := promptui.Prompt{
			Label:     promptLabel,
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	logger.Actionf("deleting components in %s namespace", *kubeconfigArgs.Namespace)
	uninstall.Components(ctx, logger, kubeClient, *kubeconfigArgs.Namespace, uninstallArgs.dryRun)

	logger.Actionf("deleting toolkit.fluxcd.io finalizers in all namespaces")
	uninstall.Finalizers(ctx, logger, kubeClient, uninstallArgs.dryRun)

	logger.Actionf("deleting toolkit.fluxcd.io custom resource definitions")
	uninstall.CustomResourceDefinitions(ctx, logger, kubeClient, uninstallArgs.dryRun)

	if !uninstallArgs.keepNamespace {
		uninstall.Namespace(ctx, logger, kubeClient, *kubeconfigArgs.Namespace, uninstallArgs.dryRun)
	}

	logger.Successf("uninstall finished")
	return nil
}
