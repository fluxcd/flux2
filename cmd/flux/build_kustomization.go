/*
Copyright 2021 The Flux authors

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
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/build"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
)

var buildKsCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"ks"},
	Short:   "Build Kustomization",
	Long: `The build command queries the Kubernetes API and fetches the specified Flux Kustomization. 
It then uses the fetched in cluster flux kustomization to perform needed transformation on the local kustomization.yaml
pointed at by --path. The local kustomization.yaml is generated if it does not exist. Finally it builds the overlays using the local kustomization.yaml, and write the resulting multi-doc YAML to stdout.

It is possible to specify a Flux kustomization file using --kustomization-file.`,
	Example: `# Build the local manifests as they were built on the cluster
flux build kustomization my-app --path ./path/to/local/manifests

# Build using a local flux kustomization file
flux build kustomization my-app --path ./path/to/local/manifests --kustomization-file ./path/to/local/my-app.yaml`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              buildKsCmdRun,
}

type buildKsFlags struct {
	kustomizationFile string
	path              string
}

var buildKsArgs buildKsFlags

func init() {
	buildKsCmd.Flags().StringVar(&buildKsArgs.path, "path", "", "Path to the manifests location.")
	buildKsCmd.Flags().StringVar(&buildKsArgs.kustomizationFile, "kustomization-file", "", "Path to the Flux Kustomization YAML file.")
	buildCmd.AddCommand(buildKsCmd)
}

func buildKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", kustomizationType.humanKind)
	}
	name := args[0]

	if buildKsArgs.path == "" {
		return fmt.Errorf("invalid resource path %q", buildKsArgs.path)
	}

	if fs, err := os.Stat(buildKsArgs.path); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid resource path %q", buildKsArgs.path)
	}

	if buildKsArgs.kustomizationFile != "" {
		if fs, err := os.Stat(buildKsArgs.kustomizationFile); os.IsNotExist(err) || fs.IsDir() {
			return fmt.Errorf("invalid kustomization file %q", buildKsArgs.kustomizationFile)
		}
	}

	builder, err := build.NewBuilder(kubeconfigArgs, kubeclientOptions, name, buildKsArgs.path, build.WithTimeout(rootArgs.timeout), build.WithKustomizationFile(buildKsArgs.kustomizationFile))
	if err != nil {
		return err
	}

	// create a signal channel
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

	errChan := make(chan error)
	go func() {
		manifests, err := builder.Build()
		if err != nil {
			errChan <- err
		}

		cmd.Print(string(manifests))
		errChan <- nil
	}()

	select {
	case <-sigc:
		fmt.Println("Build cancelled... exiting.")
		return builder.Cancel()
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	return nil

}
