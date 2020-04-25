package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the toolkit components",
	Long: `
The uninstall command removes the namespace, cluster roles,
cluster role bindings and CRDs`,
	Example: `  uninstall --namespace=gitops-system --crds --dry-run`,
	RunE:    uninstallCmdRun,
}

var (
	uninstallCRDs   bool
	uninstallDryRun bool
)

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallCRDs, "crds", "", false,
		"removes all CRDs previously installed")
	uninstallCmd.Flags().BoolVarP(&uninstallDryRun, "dry-run", "", false,
		"only print the object that would be deleted")

	rootCmd.AddCommand(uninstallCmd)
}

func uninstallCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dryRun := ""
	if uninstallDryRun {
		dryRun = "--dry-run=client"
	} else {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to delete the %s namespace", namespace),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			fmt.Println(`✗`, "aborting")
			os.Exit(1)
		}
	}

	kinds := "namespace,clusterroles,clusterrolebindings"
	if uninstallCRDs {
		kinds += ",crds"
	}

	command := fmt.Sprintf("kubectl delete %s -l app.kubernetes.io/instance=%s --timeout=%s %s",
		kinds, namespace, timeout.String(), dryRun)
	c := exec.CommandContext(ctx, "/bin/sh", "-c", command)

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	fmt.Println(`✚`, "uninstalling...")
	err := c.Run()
	if err != nil {
		fmt.Println(`✗`, "uninstall failed")
		os.Exit(1)
	}

	fmt.Println(`✔`, "uninstall finished")
	return nil
}
