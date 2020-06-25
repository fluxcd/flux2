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
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the toolkit components",
	Long:  "The uninstall command removes the namespace, cluster roles, cluster role bindings and CRDs from the cluster.",
	Example: `  # Dry-run uninstall of all components
   uninstall --dry-run --namespace=gitops-system

  # Uninstall all components and delete custom resource definitions
  uninstall --crds --namespace=gitops-system
`,
	RunE: uninstallCmdRun,
}

var (
	uninstallCRDs           bool
	uninstallKustomizations bool
	uninstallDryRun         bool
	uninstallSilent         bool
)

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallKustomizations, "kustomizations", "", false,
		"removes all Kustomizations previously installed")
	uninstallCmd.Flags().BoolVarP(&uninstallCRDs, "crds", "", false,
		"removes all CRDs previously installed")
	uninstallCmd.Flags().BoolVarP(&uninstallDryRun, "dry-run", "", false,
		"only print the object that would be deleted")
	uninstallCmd.Flags().BoolVarP(&uninstallSilent, "silent", "s", false,
		"delete components without asking for confirmation")

	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dryRun := ""
	if uninstallDryRun {
		dryRun = "--dry-run=client"
	} else if !uninstallSilent {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to delete the %s namespace", namespace),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			return fmt.Errorf("aborting")
		}
	}

	if uninstallKustomizations {
		logger.Actionf("uninstalling kustomizations")
		command := fmt.Sprintf("kubectl -n %s delete kustomizations --all --timeout=%s %s",
			namespace, timeout.String(), dryRun)
		if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
			return fmt.Errorf("uninstall failed")
		}

		// TODO: use the kustomizations snapshots to create a list of objects
		// that are subject to deletion and wait for all of them to be terminated
		logger.Waitingf("waiting on GC")
		time.Sleep(30 * time.Second)
	}

	kinds := "namespace,clusterroles,clusterrolebindings"
	if uninstallCRDs {
		kinds += ",crds"
	}

	logger.Actionf("uninstalling components")
	command := fmt.Sprintf("kubectl delete %s -l app.kubernetes.io/instance=%s --timeout=%s %s",
		kinds, namespace, timeout.String(), dryRun)
	if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
		return fmt.Errorf("uninstall failed")
	}

	logger.Successf("uninstall finished")
	return nil
}
