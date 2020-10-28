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
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fluxcd/toolkit/internal/flags"
	"github.com/fluxcd/toolkit/internal/utils"
	"github.com/fluxcd/toolkit/pkg/install"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the toolkit components",
	Long: `The install command deploys the toolkit components in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.`,
	Example: `  # Install the latest version in the gotk-system namespace
  gotk install --version=latest --namespace=gotk-system

  # Dry-run install for a specific version and a series of components
  gotk install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview
  gotk install --dry-run --verbose

  # Write install manifests to file
  gotk install --export > gotk-system.yaml
`,
	RunE: installCmdRun,
}

var (
	installExport             bool
	installDryRun             bool
	installManifestsPath      string
	installVersion            string
	installComponents         []string
	installRegistry           string
	installImagePullSecret    string
	installArch               flags.Arch = "amd64"
	installWatchAllNamespaces bool
	installNetworkPolicy      bool
	installLogLevel           flags.LogLevel = "info"
)

func init() {
	installCmd.Flags().BoolVar(&installExport, "export", false,
		"write the install manifests to stdout and exit")
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installVersion, "version", "v", defaultVersion,
		"toolkit version")
	installCmd.Flags().StringSliceVar(&installComponents, "components", defaultComponents,
		"list of components, accepts comma-separated values")
	installCmd.Flags().StringVar(&installManifestsPath, "manifests", "", "path to the manifest directory")
	installCmd.Flags().MarkHidden("manifests")
	installCmd.Flags().StringVar(&installRegistry, "registry", "ghcr.io/fluxcd",
		"container registry where the toolkit images are published")
	installCmd.Flags().StringVar(&installImagePullSecret, "image-pull-secret", "",
		"Kubernetes secret name used for pulling the toolkit images from a private registry")
	installCmd.Flags().Var(&installArch, "arch", installArch.Description())
	installCmd.Flags().BoolVar(&installWatchAllNamespaces, "watch-all-namespaces", true,
		"watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed")
	installCmd.Flags().Var(&installLogLevel, "log-level", installLogLevel.Description())
	installCmd.Flags().BoolVar(&installNetworkPolicy, "network-policy", true,
		"deny ingress access to the toolkit controllers from other namespaces using network policies")
	rootCmd.AddCommand(installCmd)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if !installExport {
		logger.Generatef("generating manifests")
	}

	opts := install.Options{
		BaseURL:                installManifestsPath,
		Version:                installVersion,
		Namespace:              namespace,
		Components:             installComponents,
		Registry:               installRegistry,
		ImagePullSecret:        installImagePullSecret,
		Arch:                   installArch.String(),
		WatchAllNamespaces:     installWatchAllNamespaces,
		NetworkPolicy:          installNetworkPolicy,
		LogLevel:               installLogLevel.String(),
		NotificationController: defaultNotification,
		ManifestsFile:          fmt.Sprintf("%s.yaml", namespace),
		Timeout:                timeout,
	}

	if installManifestsPath == "" {
		opts.BaseURL = install.MakeDefaultOptions().BaseURL
	}

	_, content, err := install.Generate(opts)
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	manifest := path.Join(tmpDir, fmt.Sprintf("%s.yaml", namespace))
	if err := ioutil.WriteFile(manifest, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	if verbose {
		fmt.Print(content)
	} else if installExport {
		fmt.Println("---")
		fmt.Println("# GitOps Toolkit revision", installVersion)
		fmt.Println("# Components:", strings.Join(installComponents, ","))
		fmt.Print(content)
		fmt.Println("---")
		return nil
	}

	logger.Successf("manifests build completed")
	logger.Actionf("installing components in %s namespace", namespace)
	applyOutput := utils.ModeStderrOS
	if verbose {
		applyOutput = utils.ModeOS
	}

	kubectlArgs := []string{"apply", "-f", manifest}
	if installDryRun {
		args = append(args, "--dry-run=client")
		applyOutput = utils.ModeOS
	}
	if _, err := utils.ExecKubectlCommand(ctx, applyOutput, kubectlArgs...); err != nil {
		return fmt.Errorf("install failed")
	}

	if installDryRun {
		logger.Successf("install dry-run finished")
		return nil
	} else {
		logger.Successf("install completed")
	}

	logger.Waitingf("verifying installation")
	for _, deployment := range installComponents {
		kubectlArgs = []string{"-n", namespace, "rollout", "status", "deployment", deployment, "--timeout", timeout.String()}
		if _, err := utils.ExecKubectlCommand(ctx, applyOutput, kubectlArgs...); err != nil {
			return fmt.Errorf("install failed")
		} else {
			logger.Successf("%s ready", deployment)
		}
	}

	logger.Successf("install finished")
	return nil
}
