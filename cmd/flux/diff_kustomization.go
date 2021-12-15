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

var diffKsCmd = &cobra.Command{
	Use:     "kustomization",
	Aliases: []string{"ks"},
	Short:   "Diff Kustomization",
	Long:    `The diff command does a build, then it performs a server-side dry-run and output the diff.`,
	Example: `# Create a new overlay.
flux diff kustomization my-app --path ./path/to/local/manifests`,
	ValidArgsFunction: resourceNamesCompletionFunc(kustomizev1.GroupVersion.WithKind(kustomizev1.KustomizationKind)),
	RunE:              diffKsCmdRun,
}

type diffKsFlags struct {
	path string
}

var diffKsArgs diffKsFlags

func init() {
	diffKsCmd.Flags().StringVar(&diffKsArgs.path, "path", "", "Name of a file containing a file to add to the kustomization file.)")
	diffCmd.AddCommand(diffKsCmd)
}

func diffKsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("%s name is required", kustomizationType.humanKind)
	}
	name := args[0]

	if diffKsArgs.path == "" {
		return fmt.Errorf("invalid resource path %q", diffKsArgs.path)
	}

	if fs, err := os.Stat(diffKsArgs.path); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid resource path %q", diffKsArgs.path)
	}

	builder, err := kustomization.NewBuilder(kubeconfigArgs, name, diffKsArgs.path, kustomization.WithTimeout(rootArgs.timeout))
	if err != nil {
		return err
	}

	output, err := builder.Diff()
	if err != nil {
		return err
	}

	cmd.Print(output)

	return nil

}
