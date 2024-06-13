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
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	runclient "github.com/fluxcd/pkg/runtime/client"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ns, err := cmd.Flags().GetString("namespace")
		if err != nil {
			return fmt.Errorf("error getting namespace: %w", err)
		}

		if e := validation.IsDNS1123Label(ns); len(e) > 0 {
			return fmt.Errorf("namespace must be a valid DNS label: %q", ns)
		}

		return nil
	},
}

var logger = stderrLogger{stderr: os.Stderr}

type rootFlags struct {
	timeout      time.Duration
	verbose      bool
	pollInterval time.Duration
	defaults     install.Options
}

// RequestError is a custom error type that wraps an error returned by the flux api.
type RequestError struct {
	StatusCode int
	Err        error
}

func (r *RequestError) Error() string {
	return r.Err.Error()
}

var rootArgs = NewRootFlags()
var kubeconfigArgs = genericclioptions.NewConfigFlags(false)
var kubeclientOptions = new(runclient.Options)

func init() {
	rootCmd.PersistentFlags().DurationVar(&rootArgs.timeout, "timeout", 5*time.Minute, "timeout for this operation")
	rootCmd.PersistentFlags().BoolVar(&rootArgs.verbose, "verbose", false, "print generated objects")

	configureDefaultNamespace()
	kubeconfigArgs.APIServer = nil // prevent AddFlags from configuring --server flag
	kubeconfigArgs.Timeout = nil   // prevent AddFlags from configuring --request-timeout flag, we have --timeout instead
	kubeconfigArgs.AddFlags(rootCmd.PersistentFlags())

	// Since some subcommands use the `-s` flag as a short version for `--silent`, we manually configure the server flag
	// without the `-s` short version. While we're no longer on par with kubectl's flags, we maintain backwards compatibility
	// on the CLI interface.
	apiServer := ""
	kubeconfigArgs.APIServer = &apiServer
	rootCmd.PersistentFlags().StringVar(kubeconfigArgs.APIServer, "server", *kubeconfigArgs.APIServer, "The address and port of the Kubernetes API server")
	// Update the description for kubeconfig TLS flags so that user's don't mistake it for a Flux specific flag
	rootCmd.Flag("insecure-skip-tls-verify").Usage = "If true, the Kubernetes API server's certificate will not be checked for validity. This will make your HTTPS connections insecure"
	rootCmd.Flag("client-certificate").Usage = "Path to a client certificate file for TLS authentication to the Kubernetes API server"
	rootCmd.Flag("certificate-authority").Usage = "Path to a cert file for the certificate authority to authenticate the Kubernetes API server"
	rootCmd.Flag("client-key").Usage = "Path to a client key file for TLS authentication to the Kubernetes API server"

	kubeclientOptions.BindFlags(rootCmd.PersistentFlags())

	rootCmd.RegisterFlagCompletionFunc("context", contextsCompletionFunc)
	rootCmd.RegisterFlagCompletionFunc("namespace", resourceNamesCompletionFunc(corev1.SchemeGroupVersion.WithKind("Namespace")))

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

	// This is required because controller-runtime expects its consumers to
	// set a logger through log.SetLogger within 30 seconds of the program's
	// initalization. If not set, the entire debug stack is printed as an
	// error, see: https://github.com/kubernetes-sigs/controller-runtime/blob/ed8be90/pkg/log/log.go#L59
	// Since we have our own logging and don't care about controller-runtime's
	// logger, we configure it's logger to do nothing.
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))

	if err := rootCmd.Execute(); err != nil {

		if err, ok := err.(*RequestError); ok {
			if err.StatusCode == 1 {
				logger.Warningf("%v", err)
			} else {
				logger.Failuref("%v", err)
			}

			os.Exit(err.StatusCode)
		}

		logger.Failuref("%v", err)
		os.Exit(1)
	}
}

func configureDefaultNamespace() {
	*kubeconfigArgs.Namespace = rootArgs.defaults.Namespace
	fromEnv := os.Getenv("FLUX_SYSTEM_NAMESPACE")
	if fromEnv != "" {
		// namespace must be a valid DNS label. Assess against validation
		// used upstream, and ignore invalid values as environment vars
		// may not be actively provided by end-user.
		if e := validation.IsDNS1123Label(fromEnv); len(e) > 0 {
			logger.Warningf(" ignoring invalid FLUX_SYSTEM_NAMESPACE: %q", fromEnv)
			return
		}

		kubeconfigArgs.Namespace = &fromEnv
	}
}

// readPasswordFromStdin reads a password from stdin and returns the input
// with trailing newline and/or carriage return removed. It also makes sure that terminal
// echoing is turned off if stdin is a terminal.
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
	return strings.TrimRight(out, "\r\n"), nil
}

func withPreviewNote(desc string) string {
	previewNote := `⚠️  Please note that this command is in preview and under development.
While we try our best to not introduce breaking changes, they may occur when
we adapt to new features and/or find better ways to facilitate what it does.`
	return fmt.Sprintf("%s\n\n%s", strings.TrimSpace(desc), previewNote)
}
