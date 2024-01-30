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

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	ssautil "github.com/fluxcd/pkg/ssa/utils"

	"github.com/fluxcd/flux2/v2/internal/build"
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
flux build kustomization my-app --path ./path/to/local/manifests --kustomization-file ./path/to/local/my-app.yaml

# Build in dry-run mode without connecting to the cluster.
# Note that variable substitutions from Secrets and ConfigMaps are skipped in dry-run mode.
flux build kustomization my-app --path ./path/to/local/manifests \
	--kustomization-file ./path/to/local/my-app.yaml \
	--dry-run

# Exclude files by providing a comma separated list of entries that follow the .gitignore pattern fromat.
flux build kustomization my-app --path ./path/to/local/manifests \
	--kustomization-file ./path/to/local/my-app.yaml \
	--ignore-paths "/to_ignore/**/*.yaml,ignore.yaml"`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              buildKsCmdRun,
}

type buildKsFlags struct {
	kustomizationFile string
	path              string
	ignorePaths       []string
	dryRun            bool
}

var buildKsArgs buildKsFlags

func init() {
	buildKsCmd.Flags().StringVar(&buildKsArgs.path, "path", "", "Path to the manifests location.")
	buildKsCmd.Flags().StringVar(&buildKsArgs.kustomizationFile, "kustomization-file", "", "Path to the Flux Kustomization YAML file.")
	buildKsCmd.Flags().StringSliceVar(&buildKsArgs.ignorePaths, "ignore-paths", nil, "set paths to ignore in .gitignore format")
	buildKsCmd.Flags().BoolVar(&buildKsArgs.dryRun, "dry-run", false, "Dry run mode.")
	buildCmd.AddCommand(buildKsCmd)
}

func buildKsCmdRun(cmd *cobra.Command, args []string) (err error) {
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

	if buildKsArgs.dryRun && buildKsArgs.kustomizationFile == "" {
		return fmt.Errorf("dry-run mode requires a kustomization file")
	}

	if buildKsArgs.kustomizationFile != "" {
		if fs, err := os.Stat(buildKsArgs.kustomizationFile); os.IsNotExist(err) || fs.IsDir() {
			return fmt.Errorf("invalid kustomization file %q", buildKsArgs.kustomizationFile)
		}
	}

	var builder *build.Builder
	if buildKsArgs.dryRun {
		builder, err = build.NewBuilder(name, buildKsArgs.path,
			build.WithTimeout(rootArgs.timeout),
			build.WithKustomizationFile(buildKsArgs.kustomizationFile),
			build.WithDryRun(buildKsArgs.dryRun),
			build.WithNamespace(*kubeconfigArgs.Namespace),
			build.WithIgnore(buildKsArgs.ignorePaths),
		)
	} else {
		builder, err = build.NewBuilder(name, buildKsArgs.path,
			build.WithClientConfig(kubeconfigArgs, kubeclientOptions),
			build.WithTimeout(rootArgs.timeout),
			build.WithKustomizationFile(buildKsArgs.kustomizationFile),
			build.WithIgnore(buildKsArgs.ignorePaths),
		)
	}

	if err != nil {
		return err
	}

	// create a signal channel
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)

	errChan := make(chan error)
	go func() {
		objects, err := builder.Build()
		if err != nil {
			errChan <- err
		}

		manifests, err := ssautil.ObjectsToYAML(objects)
		if err != nil {
			errChan <- err
		}

		cmd.Print(manifests)
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
