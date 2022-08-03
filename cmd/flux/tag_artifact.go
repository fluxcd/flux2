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
)

var tagArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Tag artifact",
	Long: `The tag artifact command creates tags for the given OCI artifact.
The tag command uses the credentials from '~/.docker/config.json'.`,
	Example: `# Tag an artifact version as latest
flux tag artifact oci://ghcr.io/org/manifests/app:v0.0.1 --tag latest
`,
	RunE: tagArtifactCmdRun,
}

type tagArtifactFlags struct {
	tags []string
}

var tagArtifactArgs tagArtifactFlags

func init() {
	tagArtifactCmd.Flags().StringSliceVar(&tagArtifactArgs.tags, "tag", nil, "Tag name.")
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

	ociClient := oci.NewLocalClient()
	url, err := oci.ParseArtifactURL(ociURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

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
