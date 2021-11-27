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

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/internal/kustomization"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
)

var buildKsCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"ks"},
	Short:   "Build Kustomization",
	Long: `The build command queries the Kubernetes API and fetches the specified Flux Kustomization, 
then it uses the specified files or path to build the overlay to write the resulting multi-doc YAML to stdout.`,
	Example: `# Create a new overlay.
flux build kustomization my-app --resources ./path/to/local/manifests`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              buildKsCmdRun,
}

type buildKsFlags struct {
	resources string
}

var buildKsArgs buildKsFlags

func init() {
	buildKsCmd.Flags().StringVar(&buildKsArgs.resources, "resources", "", "Name of a file containing a file to add to the kustomization file.)")
	buildCmd.AddCommand(buildKsCmd)
}

func buildKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", kustomizationType.humanKind)
	}
	name := args[0]

	if buildKsArgs.resources == "" {
		return fmt.Errorf("invalid resource path %q", buildKsArgs.resources)
	}

	if fs, err := os.Stat(buildKsArgs.resources); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid resource path %q", buildKsArgs.resources)
	}

	builder, err := kustomization.NewBuilder(rootArgs.kubeconfig, rootArgs.kubecontext, rootArgs.namespace, name, buildKsArgs.resources, kustomization.WithTimeout(rootArgs.timeout))
	if err != nil {
		return err
	}

	manifests, err := builder.Build()
	if err != nil {
		return err
	}

	cmd.Print(string(manifests))

	return nil

}
