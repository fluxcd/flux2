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
	"os"

	"github.com/fluxcd/flux2/internal/flags"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/spf13/cobra"

	oci "github.com/fluxcd/pkg/oci/client"
)

var pushArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Push artifact",
	Long: `The push artifact command creates a tarball from the given directory or the single file and uploads the artifact to an OCI repository.
The command can read the credentials from '~/.docker/config.json' but they can also be passed with --creds. It can also login to a supported provider with the --provider flag.`,
	Example: `  # Push manifests to GHCR using the short Git SHA as the OCI artifact tag
  echo $GITHUB_PAT | docker login ghcr.io --username flux --password-stdin
  flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)/$(git rev-parse HEAD)"

  # Push manifests passed into stdin to GHCR
  kustomize build . | flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) -p - \ 
    --source="$(git config --get remote.origin.url)" \
    --revision="$(git branch --show-current)/$(git rev-parse HEAD)"

  # Push single manifest file to GHCR using the short Git SHA as the OCI artifact tag
  echo $GITHUB_PAT | docker login ghcr.io --username flux --password-stdin
  flux push artifact oci://ghcr.io/org/config/app:$(git rev-parse --short HEAD) \
	--path="./path/to/local/manifest.yaml" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git branch --show-current)/$(git rev-parse HEAD)"

  # Push manifests to Docker Hub using the Git tag as the OCI artifact tag
  echo $DOCKER_PAT | docker login --username flux --password-stdin
  flux push artifact oci://docker.io/org/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)/$(git rev-parse HEAD)"

  # Login directly to the registry provider
  # You might need to export the following variable if you use local config files for AWS:
  # export AWS_SDK_LOAD_CONFIG=1
  flux push artifact oci://<account>.dkr.ecr.<region>.amazonaws.com/foo:v1:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)/$(git rev-parse HEAD)" \
	--provider aws

  # Or pass credentials directly
  flux push artifact oci://docker.io/org/app-config:$(git tag --points-at HEAD) \
	--path="./path/to/local/manifests" \
	--source="$(git config --get remote.origin.url)" \
	--revision="$(git tag --points-at HEAD)/$(git rev-parse HEAD)" \
	--creds flux:$DOCKER_PAT
`,
	RunE: pushArtifactCmdRun,
}

type pushArtifactFlags struct {
	path        string
	source      string
	revision    string
	creds       string
	provider    flags.SourceOCIProvider
	ignorePaths []string
}

var pushArtifactArgs = newPushArtifactFlags()

func newPushArtifactFlags() pushArtifactFlags {
	return pushArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.path, "path", "", "path to the directory where the Kubernetes manifests are located")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.source, "source", "", "the source address, e.g. the Git URL")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.revision, "revision", "", "the source revision in the format '<branch|tag>/<commit-sha>'")
	pushArtifactCmd.Flags().StringVar(&pushArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	pushArtifactCmd.Flags().Var(&pushArtifactArgs.provider, "provider", pushArtifactArgs.provider.Description())
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

	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	if _, err := os.Stat(pushArtifactArgs.path); err != nil {
		return fmt.Errorf("invalid path '%s', must point to an existing directory or file", buildArtifactArgs.path)
	}

	meta := oci.Metadata{
		Source:   pushArtifactArgs.source,
		Revision: pushArtifactArgs.revision,
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	ociClient := oci.NewLocalClient()

	if pushArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && pushArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		if err := ociClient.LoginWithCredentials(pushArtifactArgs.creds); err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
	}

	if pushArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		ociProvider, err := pushArtifactArgs.provider.ToOCIProvider()
		if err != nil {
			return fmt.Errorf("provider not supported: %w", err)
		}

		if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
	}

	logger.Actionf("pushing artifact to %s", url)

	path := pushArtifactArgs.path
	if buildArtifactArgs.path == "-" {
		path, err = saveStdinToFile()
		if err != nil {
			return err
		}

		defer os.Remove(path)
	}

	digest, err := ociClient.Push(ctx, url, path, meta, pushArtifactArgs.ignorePaths)
	if err != nil {
		return fmt.Errorf("pushing artifact failed: %w", err)
	}

	logger.Successf("artifact successfully pushed to %s", digest)

	return nil
}
