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
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

type listArtifactFlags struct {
	semverFilter string
	regexFilter  string
	creds        string
	provider     flags.SourceOCIProvider
}

var listArtifactArgs = newListArtifactFlags()

func newListArtifactFlags() listArtifactFlags {
	return listArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),
	}
}

var listArtifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "list artifacts",
	Long: withPreviewNote(`The list command fetches the tags and their metadata from a remote OCI repository.
The command can read the credentials from '~/.docker/config.json' but they can also be passed with --creds. It can also login to a supported provider with the --provider flag.`),
	Example: `  # List the artifacts stored in an OCI repository
  flux list artifact oci://ghcr.io/org/config/app
`,
	RunE: listArtifactsCmdRun,
}

func init() {
	listArtifactsCmd.Flags().StringVar(&listArtifactArgs.semverFilter, "filter-semver", "", "filter tags returned from the oci repository using semver")
	listArtifactsCmd.Flags().StringVar(&listArtifactArgs.regexFilter, "filter-regex", "", "filter tags returned from the oci repository using regex")
	listArtifactsCmd.Flags().StringVar(&listArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	listArtifactsCmd.Flags().Var(&listArtifactArgs.provider, "provider", listArtifactArgs.provider.Description())

	listCmd.AddCommand(listArtifactsCmd)
}

func listArtifactsCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact repository URL is required")
	}
	ociURL := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	ociClient := oci.NewClient(oci.DefaultOptions())

	if listArtifactArgs.provider.String() == sourcev1.GenericOCIProvider && listArtifactArgs.creds != "" {
		logger.Actionf("logging in to registry with credentials")
		if err := ociClient.LoginWithCredentials(listArtifactArgs.creds); err != nil {
			return fmt.Errorf("could not login with credentials: %w", err)
		}
	}

	if listArtifactArgs.provider.String() != sourcev1.GenericOCIProvider {
		logger.Actionf("logging in to registry with provider credentials")
		ociProvider, err := listArtifactArgs.provider.ToOCIProvider()
		if err != nil {
			return fmt.Errorf("provider not supported: %w", err)
		}

		if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
			return fmt.Errorf("error during login with provider: %w", err)
		}
	}

	opts := oci.ListOptions{
		RegexFilter:  listArtifactArgs.regexFilter,
		SemverFilter: listArtifactArgs.semverFilter,
	}

	metas, err := ociClient.List(ctx, url, opts)
	if err != nil {
		return err
	}

	var rows [][]string
	for _, meta := range metas {
		rows = append(rows, []string{meta.URL, meta.Digest, meta.Source, meta.Revision})
	}

	err = printers.TablePrinter([]string{"artifact", "digest", "source", "revision"}).Print(cmd.OutOrStdout(), rows)
	if err != nil {
		return err
	}

	return nil
}
