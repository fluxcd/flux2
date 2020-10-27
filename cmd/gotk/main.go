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
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	gotklog "github.com/fluxcd/toolkit/pkg/log"
)

var VERSION = "0.0.0-dev.0"

var rootCmd = &cobra.Command{
	Use:           "gotk",
	Version:       VERSION,
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Command line utility for assembling Kubernetes CD pipelines",
	Long:          `Command line utility for assembling Kubernetes CD pipelines the GitOps way.`,
	Example: `  # Check prerequisites
  gotk check --pre

  # Install the latest version of the toolkit
  gotk install --version=master

  # Create a source from a public Git repository
  gotk create source git webapp-latest \
    --url=https://github.com/stefanprodan/podinfo \
    --branch=master \
    --interval=3m

  # List GitRepository sources and their status
  gotk get sources git

  # Trigger a GitRepository source reconciliation
  gotk reconcile source git gotk-system

  # Export GitRepository sources in YAML format
  gotk export source git --all > sources.yaml

  # Create a Kustomization for deploying a series of microservices
  gotk create kustomization webapp-dev \
    --source=webapp-latest \
    --path="./deploy/webapp/" \
    --prune=true \
    --interval=5m \
    --validation=client \
    --health-check="Deployment/backend.webapp" \
    --health-check="Deployment/frontend.webapp" \
    --health-check-timeout=2m

  # Trigger a git sync of the Kustomization's source and apply changes
  gotk reconcile kustomization webapp-dev --with-source

  # Suspend a Kustomization reconciliation
  gotk suspend kustomization webapp-dev

  # Export Kustomizations in YAML format
  gotk export kustomization --all > kustomizations.yaml

  # Resume a Kustomization reconciliation
  gotk resume kustomization webapp-dev

  # Delete a Kustomization
  gotk delete kustomization webapp-dev

  # Delete a GitRepository source
  gotk delete source git webapp-latest

  # Uninstall the toolkit and delete CRDs
  gotk uninstall --crds
`,
}

var (
	kubeconfig   string
	namespace    string
	timeout      time.Duration
	verbose      bool
	pollInterval                = 2 * time.Second
	logger       gotklog.Logger = printLogger{}
)

var (
	defaultComponents   = []string{"source-controller", "kustomize-controller", "helm-controller", "notification-controller"}
	defaultVersion      = "latest"
	defaultNamespace    = "gotk-system"
	defaultNotification = "notification-controller"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", defaultNamespace, "the namespace scope for this operation")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 5*time.Minute, "timeout for this operation")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print generated objects")
}

func main() {
	log.SetFlags(0)
	generateDocs()
	kubeconfigFlag()
	if err := rootCmd.Execute(); err != nil {
		logger.Failuref("%v", err)
		os.Exit(1)
	}
}

func kubeconfigFlag() {
	if home := homeDir(); home != "" {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", filepath.Join(home, ".kube", "config"),
			"path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "",
			"absolute path to the kubeconfig file")
	}

	if len(os.Getenv("KUBECONFIG")) > 0 {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
}

func generateDocs() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "docgen" {
		rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "~/.kube/config",
			"path to the kubeconfig file")
		rootCmd.DisableAutoGenTag = true
		err := doc.GenMarkdownTree(rootCmd, "./docs/cmd")
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
