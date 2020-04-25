package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
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
)

func init() {
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installManifestsPath, "manifests", "", "",
		"path to the manifest directory")

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

	logAction("installing components in %s namespace", namespace)
	err := c.Run()
	if err != nil {
		logFailure("install failed")
		os.Exit(1)
	}

	if installDryRun {
		logSuccess("install dry-run finished")
		return nil
	}

	logAction("verifying installation")
	for _, deployment := range []string{"source-controller", "kustomize-controller"} {
		command = fmt.Sprintf("kubectl -n %s rollout status deployment %s --timeout=%s",
			namespace, deployment, timeout.String())
		c = exec.CommandContext(ctx, "/bin/sh", "-c", command)
		c.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
		c.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		err := c.Run()
		if err != nil {
			logFailure("install failed")
			os.Exit(1)
		}
	}

	logSuccess("install finished")
	return nil
}
