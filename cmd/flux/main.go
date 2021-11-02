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
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

var VERSION = "0.0.0-dev.0"

var rootCmd = &cobra.Command{
	Use:           "flux",
	Version:       VERSION,
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Command line utility for assembling Kubernetes CD pipelines",
	Long: `
Command line utility for assembling Kubernetes CD pipelines the GitOps way.`,
	Example: `  # Check prerequisites
  flux check --pre

  # Install the latest version of Flux
  flux install

  # Create a source for a public Git repository
  flux create source git webapp-latest \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master \
    --interval=3m

  # List GitRepository sources and their status
  flux get sources git

  # Trigger a GitRepository source reconciliation
  flux reconcile source git flux-system

  # Export GitRepository sources in YAML format
  flux export source git --all > sources.yaml

  # Create a Kustomization for deploying a series of microservices
  flux create kustomization webapp-dev \
    --source=webapp-latest \
    --path="./deploy/webapp/" \
    --prune=true \
    --interval=5m \
    --health-check="Deployment/backend.webapp" \
    --health-check="Deployment/frontend.webapp" \
    --health-check-timeout=2m

  # Trigger a git sync of the Kustomization's source and apply changes
  flux reconcile kustomization webapp-dev --with-source

  # Suspend a Kustomization reconciliation
  flux suspend kustomization webapp-dev

  # Export Kustomizations in YAML format
  flux export kustomization --all > kustomizations.yaml

  # Resume a Kustomization reconciliation
  flux resume kustomization webapp-dev

  # Delete a Kustomization
  flux delete kustomization webapp-dev

  # Delete a GitRepository source
  flux delete source git webapp-latest

  # Uninstall Flux and delete CRDs
  flux uninstall`,
}

var logger = stderrLogger{stderr: os.Stderr}

type rootFlags struct {
	kubeconfig   string
	kubecontext  string
	namespace    string
	timeout      time.Duration
	verbose      bool
	pollInterval time.Duration
	defaults     install.Options
}

var rootArgs = NewRootFlags()

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootArgs.namespace, "namespace", "n", rootArgs.defaults.Namespace,
		"the namespace scope for this operation, can be set with FLUX_SYSTEM_NAMESPACE env var")
	rootCmd.RegisterFlagCompletionFunc("namespace", resourceNamesCompletionFunc(corev1.SchemeGroupVersion.WithKind("Namespace")))

	rootCmd.PersistentFlags().DurationVar(&rootArgs.timeout, "timeout", 5*time.Minute, "timeout for this operation")
	rootCmd.PersistentFlags().BoolVar(&rootArgs.verbose, "verbose", false, "print generated objects")
	rootCmd.PersistentFlags().StringVarP(&rootArgs.kubeconfig, "kubeconfig", "", "",
		"absolute path to the kubeconfig file")

	rootCmd.PersistentFlags().StringVarP(&rootArgs.kubecontext, "context", "", "", "kubernetes context to use")
	rootCmd.RegisterFlagCompletionFunc("context", contextsCompletionFunc)

	rootCmd.DisableAutoGenTag = true
	rootCmd.SetOut(os.Stdout)
}

func NewRootFlags() rootFlags {
	rf := rootFlags{
		pollInterval: 2 * time.Second,
		defaults:     install.MakeDefaultOptions(),
	}
	rf.defaults.Version = "v" + VERSION
	return rf
}

func main() {
	log.SetFlags(0)
	configureKubeconfig()
	configureDefaultNamespace()
	if err := rootCmd.Execute(); err != nil {
		logger.Failuref("%v", err)
		os.Exit(1)
	}
}

func configureKubeconfig() {
	switch {
	case len(rootArgs.kubeconfig) > 0:
	case len(os.Getenv("KUBECONFIG")) > 0:
		rootArgs.kubeconfig = os.Getenv("KUBECONFIG")
	default:
		if home := homeDir(); len(home) > 0 {
			rootArgs.kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
}

func configureDefaultNamespace() {
	fromEnv := os.Getenv("FLUX_SYSTEM_NAMESPACE")
	if fromEnv != "" && rootArgs.namespace == rootArgs.defaults.Namespace {
		rootArgs.namespace = fromEnv
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func readPasswordFromStdin(prompt string) (string, error) {
	var out string
	var err error
	fmt.Fprint(os.Stdout, prompt)
	stdinFD := int(os.Stdin.Fd())
	if term.IsTerminal(stdinFD) {
		var inBytes []byte
		inBytes, err = term.ReadPassword(int(os.Stdin.Fd()))
		out = string(inBytes)
	} else {
		out, err = bufio.NewReader(os.Stdin).ReadString('\n')
	}
	if err != nil {
		return "", fmt.Errorf("could not read from stdin: %w", err)
	}
	fmt.Println()
	return out, nil
}
