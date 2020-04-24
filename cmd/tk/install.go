package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the toolkit components",
	Long: `
The Install command deploys the toolkit components
on the configured Kubernetes cluster in ~/.kube/config`,
	Example: `  install --manifests github.com/fluxcd/toolkit//manifests/install --dry-run`,
	RunE:    installCmdRun,
}

var (
	installDryRun        bool
	installManifestsPath string
	installNamespace     string
)

func init() {
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installManifestsPath, "manifests", "", "",
		"path to the manifest directory")
	installCmd.Flags().StringVarP(&installNamespace, "namespace", "", "gitops-system",
		"the namespace scope for this installation")

	rootCmd.AddCommand(installCmd)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	if installManifestsPath == "" {
		return fmt.Errorf("no manifests specified")
	}

	if !strings.HasPrefix(installManifestsPath, "github.com/") {
		if _, err := os.Stat(installManifestsPath); err != nil {
			return fmt.Errorf("manifests not found: %w", err)
		}
	}

	timeout := time.Minute * 5
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dryRun := ""
	if installDryRun {
		dryRun = "--dry-run=client"
	}
	command := fmt.Sprintf("kustomize build %s | kubectl apply -f- %s",
		installManifestsPath, dryRun)
	c := exec.CommandContext(ctx, "/bin/sh", "-c", command)

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	fmt.Println(`✚`, "installing...")
	err := c.Run()
	if err != nil {
		fmt.Println(`✗`, "install failed")
		os.Exit(1)
	}

	if installDryRun {
		fmt.Println(`✔`, "install dry-run finished")
		return nil
	}

	fmt.Println(`✚`, "verifying installation...")
	for _, deployment := range []string{"source-controller", "kustomize-controller"} {
		command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=2m",
			installNamespace, deployment)
		c = exec.CommandContext(ctx, "/bin/sh", "-c", command)
		c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err := c.Run()
		if err != nil {
			fmt.Println(`✗`, "install failed")
			os.Exit(1)
		}
	}

	fmt.Println(`✔`, "install finished")
	return nil
}
