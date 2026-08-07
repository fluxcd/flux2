/*
Copyright 2024 The Flux authors

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
	"io"
	"os"

	"github.com/fluxcd/pkg/envsubst"
	"github.com/fluxcd/pkg/kustomize"
	"github.com/spf13/cobra"
)

var envsubstCmd = &cobra.Command{
	Use:   "envsubst",
	Args:  cobra.NoArgs,
	Short: "envsubst substitutes the values of environment variables",
	Long: withPreviewNote(`The envsubst command substitutes the values of environment variables
in the string piped as standard input and writes the result to the standard output. This command can be used
to replicate the behavior of the Flux Kustomization post-build substitutions.`),
	Example: `  # Run env var substitutions on the kustomization build output
  export cluster_region=eu-central-1
  kustomize build . | flux envsubst

  # Run env var substitutions and error out if a variable is not set
  kustomize build . | flux envsubst --strict

  # Run env var substitutions, skipping resources with substitute disabled
  kustomize build . | flux envsubst --strict --k8s-aware
`,
	RunE: runEnvsubstCmd,
}

type envsubstFlags struct {
	strict   bool
	k8sAware bool
}

var envsubstArgs envsubstFlags

func init() {
	envsubstCmd.Flags().BoolVar(&envsubstArgs.strict, "strict", false,
		"fail if a variable without a default value is declared in the input but is missing from the environment")
	envsubstCmd.Flags().BoolVar(&envsubstArgs.k8sAware, "k8s-aware", false,
		"treat the input as multi-doc Kubernetes YAML and skip substitution for resources "+
			"annotated or labeled with kustomize.toolkit.fluxcd.io/substitute: disabled")
	rootCmd.AddCommand(envsubstCmd)
}

func runEnvsubstCmd(cmd *cobra.Command, args []string) error {
	data, err := io.ReadAll(rootCmd.InOrStdin())
	if err != nil {
		return err
	}

	mapping := envsubst.Getenv
	if envsubstArgs.strict {
		mapping = os.LookupEnv
	}

	var result string
	if envsubstArgs.k8sAware {
		result, err = kustomize.SubstituteEnvVariables(string(data), mapping)
	} else {
		result, err = envsubst.Eval(string(data), mapping)
	}
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(rootCmd.OutOrStdout(), result)
	return err
}
