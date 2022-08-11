/*
Copyright 2022 The Flux authors

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

	oci "github.com/fluxcd/pkg/oci/client"
)

var buildArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Build artifact",
	Long:  `The build artifact command creates a tgz file with the manifests from the given directory.`,
	Example: `  # Build the given manifests directory into an artifact
  flux build artifact --path ./path/to/local/manifests --output ./path/to/artifact.tgz

  # List the files bundled in the artifact
  tar -ztvf ./path/to/artifact.tgz
`,
	RunE: buildArtifactCmdRun,
}

type buildArtifactFlags struct {
	output string
	path   string
}

var buildArtifactArgs buildArtifactFlags

func init() {
	buildArtifactCmd.Flags().StringVar(&buildArtifactArgs.path, "path", "", "Path to the directory where the Kubernetes manifests are located.")
	buildArtifactCmd.Flags().StringVarP(&buildArtifactArgs.output, "output", "o", "artifact.tgz", "Path to where the artifact tgz file should be written.")
	buildCmd.AddCommand(buildArtifactCmd)
}

func buildArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if buildArtifactArgs.path == "" {
		return fmt.Errorf("invalid path %q", buildArtifactArgs.path)
	}

	if fs, err := os.Stat(buildArtifactArgs.path); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid path '%s', must point to an existing directory", buildArtifactArgs.path)
	}

	logger.Actionf("building artifact from %s", buildArtifactArgs.path)

	ociClient := oci.NewLocalClient()
	if err := ociClient.Build(buildArtifactArgs.output, buildArtifactArgs.path); err != nil {
		return fmt.Errorf("bulding artifact failed, error: %w", err)
	}

	logger.Successf("artifact created at %s", buildArtifactArgs.output)

	return nil
}
