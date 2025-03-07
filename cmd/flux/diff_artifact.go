/*
Copyright 2023 The Flux authors

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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	oci "github.com/fluxcd/pkg/oci/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/google/go-containerregistry/pkg/crane"

	"github.com/sergi/go-diff/diffmatchpatch"
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
	compareTo   string // New field to capture the URL or local path for comparison
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
	diffArtifactCmd.Flags().BoolP("verbose", "v", false, "Show verbose output") // Add a --verbose flag specific to diff artifact.
	diffCmd.AddCommand(diffArtifactCmd)
}

func diffArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if diffArtifactArgs.path == "" {
		return fmt.Errorf("path is required")
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

	ociContents, err := fetchContentsFromURL(ctx, url)
	if err != nil {
		return err
	}

	compareToContents, err := fetchContentsFromLocalPath(diffArtifactArgs.path)
	if err != nil {
		return err
	}

	diffOutput, err := diffContents(ociContents, compareToContents)
	if err != nil {
		return err
	}

	fmt.Printf("Diff between OCI artifact (%s) and local path (%s):\n", ociURL, diffArtifactArgs.path)

	// Show verbose output if the --verbose flag is set
	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		fmt.Println("Printing diff...")
	}

	fmt.Println(diffOutput)

	return nil
}

func fetchContentsFromURL(ctx context.Context, url string) ([]byte, error) {
	img, err := crane.Pull(url)
	if err != nil {
		return nil, fmt.Errorf("failed to pull OCI artifact: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("failed to list layers: %w", err)
	}

	if len(layers) < 1 {
		return nil, fmt.Errorf("no layers found in OCI artifact")
	}

	// Create a buffer to store the contents of the artifact
	var contentsBuffer bytes.Buffer

	// Extract the content of the first layer
	layerContent, err := layers[0].Compressed()
	if err != nil {
		return nil, fmt.Errorf("failed to extract layer content: %w", err)
	}
	defer layerContent.Close()

	// Use tar.NewReader to read the contents from the OCI artifact
	tarReader := tar.NewReader(layerContent)
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Read the file contents and write them to the buffer
		if _, err := io.Copy(&contentsBuffer, tarReader); err != nil {
			return nil, fmt.Errorf("failed to read tar entry contents: %w", err)
		}
	}

	return contentsBuffer.Bytes(), nil
}

func fetchContentsFromLocalPath(path string) ([]byte, error) {
	// Open the local file or directory at the given path
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file/directory: %w", err)
	}
	defer file.Close()

	// Create a buffer to store the contents
	var contentsBuffer bytes.Buffer

	// Use tar.NewReader to read the contents from the file/directory
	tarReader := tar.NewReader(file)
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		// Read the file contents and write them to the buffer
		if _, err := io.Copy(&contentsBuffer, tarReader); err != nil {
			return nil, fmt.Errorf("failed to read tar entry contents: %w", err)
		}
	}

	return contentsBuffer.Bytes(), nil
}

func diffContents(contents1, contents2 []byte) (string, error) {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(string(contents1), string(contents2), false)
	return dmp.DiffPrettyText(diffs), nil
}
