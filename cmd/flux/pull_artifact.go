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

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"

	"github.com/fluxcd/pkg/oci"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	"github.com/fluxcd/flux2/v2/internal/flags"
)

var pullArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Pull artifact",
	Long: `The pull artifact command downloads and extracts the OCI artifact content to the given path.
The command can read the credentials from '~/.docker/config.json' but they can also be passed with --creds. It can also login to a supported provider with the --provider flag.`,
	Example: `  # Pull an OCI artifact created by flux from GHCR
  flux pull artifact oci://ghcr.io/org/manifests/app:v0.0.1 --output ./path/to/local/manifests
`,
	RunE: pullArtifactCmdRun,
}

type pullArtifactFlags struct {
	output   string
	creds    string
	insecure bool
	provider flags.SourceOCIProvider
}

var pullArtifactArgs = newPullArtifactFlags()

func newPullArtifactFlags() pullArtifactFlags {
	return pullArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

func init() {
	pullArtifactCmd.Flags().StringVarP(&pullArtifactArgs.output, "output", "o", "", "path where the artifact content should be extracted.")
	pullArtifactCmd.Flags().StringVar(&pullArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	pullArtifactCmd.Flags().Var(&pullArtifactArgs.provider, "provider", sourceOCIRepositoryArgs.provider.Description())
	pullArtifactCmd.Flags().BoolVar(&pullArtifactArgs.insecure, "insecure-registry", false, "allows artifacts to be pulled without TLS")
	pullCmd.AddCommand(pullArtifactCmd)
}

func pullArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if pullArtifactArgs.output == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	if fs, err := os.Stat(pullArtifactArgs.output); err != nil || !fs.IsDir() {
		return fmt.Errorf("invalid output path %q: %w", pullArtifactArgs.output, err)
	}

	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	opts := oci.DefaultOptions()

	if pullArtifactArgs.insecure {
		opts = append(opts, crane.Insecure)
	}

	if pullArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		opt, _, err := loginWithProvider(ctx, url, pullArtifactArgs.provider.String())
		if err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
		opts = append(opts, opt)
	}

	ociClient := oci.NewClient(opts)

	if pullArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && pullArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		if err := ociClient.LoginWithCredentials(pullArtifactArgs.creds); err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
	}

	logger.Actionf("pulling artifact from %s", url)

	meta, err := ociClient.Pull(ctx, url, pullArtifactArgs.output)
	if err != nil {
		return err
	}

	if meta.Source != "" {
		logger.Successf("source %s", meta.Source)
	}
	if meta.Revision != "" {
		logger.Successf("revision %s", meta.Revision)
	}
	logger.Successf("digest %s", meta.Digest)
	logger.Successf("artifact content extracted to %s", pullArtifactArgs.output)

	return nil
}
