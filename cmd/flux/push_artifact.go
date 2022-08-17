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
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"

	oci "github.com/fluxcd/pkg/oci/client"
)

var pushArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Push artifact",
	Long: `The push artifact command creates a tarball from the given directory and uploads the artifact to an OCI repository.
The command uses the credentials from '~/.docker/config.json'.`,
	Example: `  # Push manifests to GHCR using the short Git SHA as the OCI artifact tag
  echo $GITHUB_PAT | docker login ghcr.io --username flux --password-stdin
  flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)/$(git rev-parse HEAD)"

  # Push manifests to Docker Hub using the Git tag as the OCI artifact tag
  echo $DOCKER_PAT | docker login --username flux --password-stdin
  flux push artifact oci://docker.io/org/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)/$(git rev-parse HEAD)"
`,
	RunE: pushArtifactCmdRun,
}

type pushArtifactFlags struct {
	path        string
	source      string
	revision    string
	ignorePaths []string
}

var pushArtifactArgs pushArtifactFlags

func init() {
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.path, "path", "", "path to the directory where the Kubernetes manifests are located")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.source, "source", "", "the source address, e.g. the Git URL")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.revision, "revision", "", "the source revision in the format '<branch|tag>/<commit-sha>'")
	pushArtifactCmd.Flags().StringSliceVar(&pushArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")

	pushCmd.AddCommand(pushArtifactCmd)
}

func pushArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if pushArtifactArgs.source == "" {
		return fmt.Errorf("--source is required")
	}

	if pushArtifactArgs.revision == "" {
		return fmt.Errorf("--revision is required")
	}

	if pushArtifactArgs.path == "" {
		return fmt.Errorf("invalid path %q", pushArtifactArgs.path)
	}

	ociClient := oci.NewLocalClient()
	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	if fs, err := os.Stat(pushArtifactArgs.path); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid path %q", pushArtifactArgs.path)
	}

	meta := oci.Metadata{
		Source:   pushArtifactArgs.source,
		Revision: pushArtifactArgs.revision,
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	logger.Actionf("pushing artifact to %s", url)

	digest, err := ociClient.Push(ctx, url, pushArtifactArgs.path, meta, pushArtifactArgs.ignorePaths)
	if err != nil {
		return fmt.Errorf("pushing artifact failed: %w", err)
	}

	logger.Successf("artifact successfully pushed to %s", digest)

	return nil
}
