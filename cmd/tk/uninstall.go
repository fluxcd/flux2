package main

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the toolkit components",
	Long: `
The uninstall command removes the namespace, cluster roles,
cluster role bindings and CRDs.`,
	Example: `  # Dry-run uninstall of all components
   uninstall --dry-run --namespace=gitops-system

  # Uninstall all components and custom resource definitions
  uninstall --crds --namespace=gitops-system
`,
	RunE: uninstallCmdRun,
}

var (
	uninstallCRDs   bool
	uninstallDryRun bool
	uninstallSilent bool
)

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallCRDs, "crds", "", false,
		"removes all CRDs previously installed")
	uninstallCmd.Flags().BoolVarP(&uninstallDryRun, "dry-run", "", false,
		"only print the object that would be deleted")
	uninstallCmd.Flags().BoolVarP(&uninstallSilent, "silent", "", false,
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

	kinds := "namespace,clusterroles,clusterrolebindings"
	if uninstallCRDs {
		kinds += ",crds"
	}

	logAction("uninstalling components")
	command := fmt.Sprintf("kubectl delete %s -l app.kubernetes.io/instance=%s --timeout=%s %s",
		kinds, namespace, timeout.String(), dryRun)
	if _, err := utils.execCommand(ctx, ModeOS, command); err != nil {
		return fmt.Errorf("uninstall failed")
	}

	logSuccess("uninstall finished")
	return nil
}
