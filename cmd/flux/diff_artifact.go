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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	oci "github.com/fluxcd/pkg/oci/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var ErrDiffArtifactChanged = errors.New("the remote and local artifact contents differ")

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
	brief       bool
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
	diffArtifactCmd.Flags().BoolVarP(&diffArtifactArgs.brief, "brief", "q", false, "Just print a line when the resources differ. Does not output a list of changes.")
	diffCmd.AddCommand(diffArtifactCmd)
}

func diffArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	ociURL := args[0]

	if diffArtifactArgs.path == "" {
		return errors.New("the '--path' flag is required")
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

	diff, err := diffArtifact(ctx, ociClient, url, diffArtifactArgs.path)
	if err != nil {
		return err
	}

	if diff == "" {
		logger.Successf("no changes detected")
		return nil
	}

	if !diffArtifactArgs.brief {
		fmt.Print(diff)
	}

	return fmt.Errorf("%q and %q: %w", ociURL, diffArtifactArgs.path, ErrDiffArtifactChanged)
}

func diffArtifact(ctx context.Context, client *oci.Client, remoteURL, localPath string) (string, error) {
	localFile, err := loadLocal(localPath)
	if err != nil {
		return "", err
	}

	remoteFile, cleanup, err := loadRemote(ctx, client, remoteURL)
	if err != nil {
		return "", err
	}
	defer cleanup()

	report, err := dyff.CompareInputFiles(remoteFile, localFile,
		dyff.IgnoreOrderChanges(false),
		dyff.KubernetesEntityDetection(true),
	)
	if err != nil {
		return "", fmt.Errorf("dyff.CompareInputFiles(): %w", err)
	}

	if len(report.Diffs) == 0 {
		return "", nil
	}

	var buf bytes.Buffer

	if err := printers.NewDyffPrinter().Print(&buf, report); err != nil {
		return "", fmt.Errorf("formatting dyff report: %w", err)
	}

	return buf.String(), nil
}

func loadLocal(path string) (ytbx.InputFile, error) {
	if ytbx.IsStdin(path) {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return ytbx.InputFile{}, fmt.Errorf("os.ReadAll(os.Stdin): %w", err)
		}

		nodes, err := ytbx.LoadDocuments(buf)
		if err != nil {
			return ytbx.InputFile{}, fmt.Errorf("ytbx.LoadDocuments(): %w", err)
		}

		return ytbx.InputFile{
			Location:  "STDIN",
			Documents: nodes,
		}, nil
	}

	sb, err := os.Stat(path)
	if err != nil {
		return ytbx.InputFile{}, fmt.Errorf("os.Stat(%q): %w", path, err)
	}

	if sb.IsDir() {
		return ytbx.LoadDirectory(path)
	}

	return ytbx.LoadFile(path)
}

func loadRemote(ctx context.Context, client *oci.Client, url string) (ytbx.InputFile, func(), error) {
	noopCleanup := func() {}

	tmpDir, err := os.MkdirTemp("", "flux-diff-artifact")
	if err != nil {
		return ytbx.InputFile{}, noopCleanup, fmt.Errorf("could not create temporary directory: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "os.RemoveAll(%q): %v\n", tmpDir, err)
		}
	}

	if _, err := client.Pull(ctx, url, tmpDir); err != nil {
		cleanup()
		return ytbx.InputFile{}, noopCleanup, fmt.Errorf("Pull(%q): %w", url, err)
	}

	inputFile, err := ytbx.LoadDirectory(tmpDir)
	if err != nil {
		cleanup()
		return ytbx.InputFile{}, noopCleanup, fmt.Errorf("ytbx.LoadDirectory(%q): %w", tmpDir, err)
	}

	inputFile.Location = url

	return inputFile, cleanup, nil
}
