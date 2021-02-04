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
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/flags"
	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the toolkit components",
	Long: `The install command deploys the toolkit components in the specified namespace.
If a previous version is installed, then an in-place upgrade will be performed.`,
	Example: `  # Install the latest version in the flux-system namespace
  flux install --version=latest --namespace=flux-system

  # Dry-run install for a specific version and a series of components
  flux install --dry-run --version=v0.0.7 --components="source-controller,kustomize-controller"

  # Dry-run install with manifests preview
  flux install --dry-run --verbose

  # Write install manifests to file
  flux install --export > flux-system.yaml
`,
	RunE: installCmdRun,
}

var (
	installExport             bool
	installDryRun             bool
	installManifestsPath      string
	installVersion            string
	installDefaultComponents  []string
	installExtraComponents    []string
	installRegistry           string
	installImagePullSecret    string
	installWatchAllNamespaces bool
	installNetworkPolicy      bool
	installArch               flags.Arch
	installLogLevel           = flags.LogLevel(rootArgs.defaults.LogLevel)
	installClusterDomain      string
)

func init() {
	installCmd.Flags().BoolVar(&installExport, "export", false,
		"write the install manifests to stdout and exit")
	installCmd.Flags().BoolVarP(&installDryRun, "dry-run", "", false,
		"only print the object that would be applied")
	installCmd.Flags().StringVarP(&installVersion, "version", "v", rootArgs.defaults.Version,
		"toolkit version")
	installCmd.Flags().StringSliceVar(&installDefaultComponents, "components", rootArgs.defaults.Components,
		"list of components, accepts comma-separated values")
	installCmd.Flags().StringSliceVar(&installExtraComponents, "components-extra", nil,
		"list of components in addition to those supplied or defaulted, accepts comma-separated values")
	installCmd.Flags().StringVar(&installManifestsPath, "manifests", "", "path to the manifest directory")
	installCmd.Flags().StringVar(&installRegistry, "registry", rootArgs.defaults.Registry,
		"container registry where the toolkit images are published")
	installCmd.Flags().StringVar(&installImagePullSecret, "image-pull-secret", "",
		"Kubernetes secret name used for pulling the toolkit images from a private registry")
	installCmd.Flags().Var(&installArch, "arch", installArch.Description())
	installCmd.Flags().BoolVar(&installWatchAllNamespaces, "watch-all-namespaces", rootArgs.defaults.WatchAllNamespaces,
		"watch for custom resources in all namespaces, if set to false it will only watch the namespace where the toolkit is installed")
	installCmd.Flags().Var(&installLogLevel, "log-level", installLogLevel.Description())
	installCmd.Flags().BoolVar(&installNetworkPolicy, "network-policy", rootArgs.defaults.NetworkPolicy,
		"deny ingress access to the toolkit controllers from other namespaces using network policies")
	installCmd.Flags().StringVar(&installClusterDomain, "cluster-domain", rootArgs.defaults.ClusterDomain, "internal cluster domain")
	installCmd.Flags().MarkHidden("manifests")
	installCmd.Flags().MarkDeprecated("arch", "multi-arch container image is now available for AMD64, ARMv7 and ARM64")
	rootCmd.AddCommand(installCmd)
}

func installCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", rootArgs.namespace)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if !installExport {
		logger.Generatef("generating manifests")
	}

	components := append(installDefaultComponents, installExtraComponents...)

	if err := utils.ValidateComponents(components); err != nil {
		return err
	}

	opts := install.Options{
		BaseURL:                installManifestsPath,
		Version:                installVersion,
		Namespace:              rootArgs.namespace,
		Components:             components,
		Registry:               installRegistry,
		ImagePullSecret:        installImagePullSecret,
		WatchAllNamespaces:     installWatchAllNamespaces,
		NetworkPolicy:          installNetworkPolicy,
		LogLevel:               installLogLevel.String(),
		NotificationController: rootArgs.defaults.NotificationController,
		ManifestFile:           fmt.Sprintf("%s.yaml", rootArgs.namespace),
		Timeout:                rootArgs.timeout,
		ClusterDomain:          installClusterDomain,
	}

	if installManifestsPath == "" {
		opts.BaseURL = install.MakeDefaultOptions().BaseURL
	}

	manifest, err := install.Generate(opts)
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	if _, err := manifest.WriteFile(tmpDir); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	if rootArgs.verbose {
		fmt.Print(manifest.Content)
	} else if installExport {
		fmt.Println("---")
		fmt.Println("# GitOps Toolkit revision", installVersion)
		fmt.Println("# Components:", strings.Join(components, ","))
		fmt.Print(manifest.Content)
		fmt.Println("---")
		return nil
	}

	logger.Successf("manifests build completed")
	logger.Actionf("installing components in %s namespace", rootArgs.namespace)
	applyOutput := utils.ModeStderrOS
	if rootArgs.verbose {
		applyOutput = utils.ModeOS
	}

	kubectlArgs := []string{"apply", "-f", filepath.Join(tmpDir, manifest.Path)}
	if installDryRun {
		kubectlArgs = append(kubectlArgs, "--dry-run=client")
		applyOutput = utils.ModeOS
	}
	if _, err := utils.ExecKubectlCommand(ctx, applyOutput, rootArgs.kubeconfig, rootArgs.kubecontext, kubectlArgs...); err != nil {
		return fmt.Errorf("install failed")
	}

	if installDryRun {
		logger.Successf("install dry-run finished")
		return nil
	}

	statusChecker, err := NewStatusChecker(time.Second, time.Minute)
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	logger.Waitingf("verifying installation")
	if err := statusChecker.Assess(components...); err != nil {
		return fmt.Errorf("install failed")
	}

	logger.Successf("install finished")
	return nil
}
