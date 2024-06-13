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

	oci "github.com/fluxcd/pkg/oci/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/flux2/v2/internal/flags"
)

var tagArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Tag artifact",
	Long: withPreviewNote(`The tag artifact command creates tags for the given OCI artifact.
The command can read the credentials from '~/.docker/config.json' but they can also be passed with --creds. It can also login to a supported provider with the --provider flag.`),
	Example: `  # Tag an artifact version as latest
  flux tag artifact oci://ghcr.io/org/manifests/app:v0.0.1 --tag latest
`,
	RunE: tagArtifactCmdRun,
}

type tagArtifactFlags struct {
	tags     []string
	creds    string
	provider flags.SourceOCIProvider
}

var tagArtifactArgs = newTagArtifactFlags()

func newTagArtifactFlags() tagArtifactFlags {
	return tagArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	tagArtifactCmd.Flags().StringSliceVar(&tagArtifactArgs.tags, "tag", nil, "tag name")
	tagArtifactCmd.Flags().StringVar(&tagArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	tagArtifactCmd.Flags().Var(&tagArtifactArgs.provider, "provider", tagArtifactArgs.provider.Description())
	tagCmd.AddCommand(tagArtifactCmd)
}

func tagArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact name is required")
	}
	ociURL := args[0]

	if len(tagArtifactArgs.tags) < 1 {
		return fmt.Errorf("--tag is required")
	}

	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	ociClient := oci.NewClient(oci.DefaultOptions())

	if tagArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && tagArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		if err := ociClient.LoginWithCredentials(tagArtifactArgs.creds); err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
	}

	if tagArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		ociProvider, err := tagArtifactArgs.provider.ToOCIProvider()
		if err != nil {
			return fmt.Errorf("provider not supported: %w", err)
		}

		if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
	}

	logger.Actionf("tagging artifact")

	for _, tag := range tagArtifactArgs.tags {
		img, err := ociClient.Tag(ctx, url, tag)
		if err != nil {
			return fmt.Errorf("tagging artifact failed: %w", err)
		}

		logger.Successf("artifact tagged as %s", img)
	}

	return nil

}
