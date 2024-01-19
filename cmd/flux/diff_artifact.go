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

	oci "github.com/fluxcd/pkg/oci/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/flags"
)

var diffArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Diff Artifact",
	Long:  withPreviewNote(`The diff artifact command computes the diff between the remote OCI artifact and a local directory or file`),
	Example: `# Check if local files differ from remote
flux diff artifact oci://ghcr.io/stefanprodan/manifests:podinfo:6.2.0 --path=./kustomize`,
	RunE: diffArtifactCmdRun,
}

type diffArtifactFlags struct {
	path        string
	creds       string
	provider    flags.SourceOCIProvider
	ignorePaths []string
}

var diffArtifactArgs = newDiffArtifactArgs()

func newDiffArtifactArgs() diffArtifactFlags {
	return diffArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.path, "path", "", "path to the directory where the Kubernetes manifests are located")
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	diffArtifactCmd.Flags().Var(&diffArtifactArgs.provider, "provider", sourceOCIRepositoryArgs.provider.Description())
	diffArtifactCmd.Flags().StringSliceVar(&diffArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")
	diffCmd.AddCommand(diffArtifactCmd)
}

func diffArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if diffArtifactArgs.path == "" {
		return fmt.Errorf("invalid path %q", diffArtifactArgs.path)
	}

	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	if _, err := os.Stat(diffArtifactArgs.path); err != nil {
		return fmt.Errorf("invalid path '%s', must point to an existing directory or file", diffArtifactArgs.path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	ociClient := oci.NewClient(oci.DefaultOptions())

	if diffArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && diffArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		if err := ociClient.LoginWithCredentials(diffArtifactArgs.creds); err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
	}

	if diffArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		ociProvider, err := diffArtifactArgs.provider.ToOCIProvider()
		if err != nil {
			return fmt.Errorf("provider not supported: %w", err)
		}

		if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
	}

	if err := ociClient.Diff(ctx, url, diffArtifactArgs.path, diffArtifactArgs.ignorePaths); err != nil {
		return err
	}

	logger.Successf("no changes detected")
	return nil
}
