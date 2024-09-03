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
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	oci "github.com/fluxcd/pkg/oci/client"
	"github.com/fluxcd/pkg/tar"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/gonvenience/ytbx"
	"github.com/google/shlex"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"github.com/fluxcd/flux2/v2/internal/flags"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var ErrDiffArtifactChanged = errors.New("the artifact contents differ")

var diffArtifactCmd = &cobra.Command{
	Use:   "artifact <from> <to>",
	Short: "Diff Artifact",
	Long: withPreviewNote(fmt.Sprintf(
		"The diff artifact command prints the diff between the remote OCI artifact and a local directory or file.\n\n"+
			"You can overwrite the command used for diffing by setting the %q environment variable.", externalDiffVar)),
	Example: `# Check if local files differ from remote
flux diff artifact oci://ghcr.io/stefanprodan/manifests:podinfo:6.2.0 ./kustomize`,
	RunE: diffArtifactCmdRun,
	Args: cobra.RangeArgs(1, 2),
}

type diffArtifactFlags struct {
	path        string
	creds       string
	provider    flags.SourceOCIProvider
	ignorePaths []string
	brief       bool
	differ      *semanticDiffFlag
}

var diffArtifactArgs = newDiffArtifactArgs()

func newDiffArtifactArgs() diffArtifactFlags {
	defaultDiffer := mustExternalDiff()

	return diffArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),

		differ: &semanticDiffFlag{
			options: map[string]differ{
				"yaml": dyffBuiltin{
					opts: []dyff.CompareOption{
						dyff.IgnoreOrderChanges(false),
						dyff.KubernetesEntityDetection(true),
					},
				},
				"false": defaultDiffer,
			},
			value:  "false",
			differ: defaultDiffer,
		},
	}
}

func init() {
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.path, "path", "", "path to the directory or file containing the Kubernetes manifests (deprecated, use a second positional argument instead)")
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	diffArtifactCmd.Flags().Var(&diffArtifactArgs.provider, "provider", sourceOCIRepositoryArgs.provider.Description())
	diffArtifactCmd.Flags().StringSliceVar(&diffArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")
	diffArtifactCmd.Flags().BoolVarP(&diffArtifactArgs.brief, "brief", "q", false, "just print a line when the resources differ; does not output a list of changes")
	diffArtifactCmd.Flags().Var(diffArtifactArgs.differ, "semantic-diff", "use a semantic diffing algorithm")

	diffCmd.AddCommand(diffArtifactCmd)
}

func diffArtifactCmdRun(cmd *cobra.Command, args []string) error {
	var from, to string

	if len(args) < 1 {
		return fmt.Errorf("artifact URL is required")
	}
	from = args[0]

	switch {
	case len(args) >= 2:
		to = args[1]

	case diffArtifactArgs.path != "":
		// for backwards compatibility
		to = diffArtifactArgs.path

	default:
		return errors.New("a second artifact is required")
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

		if url, err := oci.ParseArtifactURL(from); err == nil {
			if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
				return fmt.Errorf("error during login with provider: %w", err)
			}
		}

		if url, err := oci.ParseArtifactURL(to); err == nil {
			if err := ociClient.LoginWithProvider(ctx, url, ociProvider); err != nil {
				return fmt.Errorf("error during login with provider: %w", err)
			}
		}
	}

	diff, err := diffArtifact(ctx, ociClient, from, to, diffArtifactArgs)
	if err != nil {
		return err
	}

	if diff == "" {
		logger.Successf("no changes detected")
		return nil
	}

	if !diffArtifactArgs.brief {
		cmd.Print(diff)
	}

	return fmt.Errorf("%q and %q: %w", from, to, ErrDiffArtifactChanged)
}

func diffArtifact(ctx context.Context, client *oci.Client, from, to string, flags diffArtifactFlags) (string, error) {
	fromDir, fromCleanup, err := loadArtifact(ctx, client, from)
	if err != nil {
		return "", err
	}
	defer fromCleanup()

	toDir, toCleanup, err := loadArtifact(ctx, client, to)
	if err != nil {
		return "", err
	}
	defer toCleanup()

	return flags.differ.Diff(ctx, fromDir, toDir)
}

// loadArtifact ensures that the artifact is in a local directory that can be
// recursively diffed. If necessary, files are downloaded, extracted, and/or
// copied into temporary directories for this purpose.
func loadArtifact(ctx context.Context, client *oci.Client, path string) (dir string, cleanup func(), err error) {
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return path, func() {}, nil
	}

	if err == nil && fi.Mode().IsRegular() {
		return loadArtifactFile(path)
	}

	url, err := oci.ParseArtifactURL(path)
	if err == nil {
		return loadArtifactOCI(ctx, client, url)
	}

	return "", nil, fmt.Errorf("%q: %w", path, os.ErrNotExist)
}

// loadArtifactOCI pulls the remove artifact into a temporary directory.
func loadArtifactOCI(ctx context.Context, client *oci.Client, url string) (dir string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "flux-diff-artifact")
	if err != nil {
		return "", nil, fmt.Errorf("could not create temporary directory: %w", err)
	}

	cleanup = func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "os.RemoveAll(%q): %v\n", tmpDir, err)
		}
	}

	if _, err := client.Pull(ctx, url, tmpDir); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("Pull(%q): %w", url, err)
	}

	return tmpDir, cleanup, nil
}

// loadArtifactFile copies a file into a temporary directory to allow for recursive diffing.
// If path is a .tar.gz or .tgz file, the archive is extracted into a temporary directory.
// Otherwise the file is copied verbatim.
func loadArtifactFile(path string) (dir string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "flux-diff-artifact")
	if err != nil {
		return "", nil, fmt.Errorf("could not create temporary directory: %w", err)
	}

	cleanup = func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "os.RemoveAll(%q): %v\n", tmpDir, err)
		}
	}

	if strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz") {
		if err := extractTo(path, tmpDir); err != nil {
			cleanup()
			return "", nil, err
		}
	} else {
		fh, err := os.Open(path)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("os.Open(%q): %w", path, err)
		}
		defer fh.Close()

		name := filepath.Join(tmpDir, filepath.Base(path))
		if err := copyFile(fh, name); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("os.Open(%q): %w", path, err)
		}
	}

	return tmpDir, cleanup, nil
}

// extractTo extracts the .tar.gz / .tgz archive at archivePath into the destDir directory.
func extractTo(archivePath, destDir string) error {
	archiveFH, err := os.Open(archivePath)
	if err != nil {
		return err
	}

	if err := tar.Untar(archiveFH, destDir); err != nil {
		return fmt.Errorf("Untar(%q, %q): %w", archivePath, destDir, err)
	}

	return nil
}

func copyFile(from io.Reader, to string) error {
	fh, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("os.Create(%q): %w", to, err)
	}
	defer fh.Close()

	if _, err := io.Copy(fh, from); err != nil {
		return fmt.Errorf("io.Copy(%q): %w", to, err)
	}

	return nil
}

type differ interface {
	// Diff compares the two local directories "to" and "from" and returns their differences, or an empty string if they are equal.
	Diff(ctx context.Context, from, to string) (string, error)
}

// externalDiffCommand implements the differ interface using an external diff command.
type externalDiffCommand struct {
	name  string
	flags []string
}

// externalDiffVar is the environment variable users can use to overwrite the external diff command.
const externalDiffVar = "FLUX_EXTERNAL_DIFF"

// mustExternalDiff initializes an externalDiffCommand using the externalDiffVar environment variable.
func mustExternalDiff() externalDiffCommand {
	cmdline := os.Getenv(externalDiffVar)
	if cmdline == "" {
		cmdline = "diff -ur"
	}

	args, err := shlex.Split(cmdline)
	if err != nil {
		panic(fmt.Sprintf("shlex.Split(%q): %v", cmdline, err))
	}

	return externalDiffCommand{
		name:  args[0],
		flags: args[1:],
	}
}

func (c externalDiffCommand) Diff(ctx context.Context, fromDir, toDir string) (string, error) {
	var args []string

	args = append(args, c.flags...)
	args = append(args, fromDir, toDir)

	cmd := exec.CommandContext(ctx, c.name, args...)

	var stdout bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// exit code 1 only means there was a difference => ignore
	} else if err != nil {
		return "", fmt.Errorf("executing %q: %w", c.name, err)
	}

	return stdout.String(), nil
}

// dyffBuiltin implements the differ interface using `dyff`, a semantic diff for YAML documents.
type dyffBuiltin struct {
	opts []dyff.CompareOption
}

func (d dyffBuiltin) Diff(ctx context.Context, fromDir, toDir string) (string, error) {
	fromFile, err := ytbx.LoadDirectory(fromDir)
	if err != nil {
		return "", fmt.Errorf("ytbx.LoadDirectory(%q): %w", fromDir, err)
	}

	toFile, err := ytbx.LoadDirectory(toDir)
	if err != nil {
		return "", fmt.Errorf("ytbx.LoadDirectory(%q): %w", toDir, err)
	}

	report, err := dyff.CompareInputFiles(fromFile, toFile, d.opts...)
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

// semanticDiffFlag implements pflag.Value for choosing a semantic diffing algorithm.
type semanticDiffFlag struct {
	options map[string]differ
	value   string
	differ
}

func (f *semanticDiffFlag) Set(s string) error {
	d, ok := f.options[s]
	if !ok {
		return fmt.Errorf("invalid value: %q", s)
	}

	f.value = s
	f.differ = d

	return nil
}

func (f *semanticDiffFlag) String() string {
	return f.value
}

func (f *semanticDiffFlag) Type() string {
	keys := maps.Keys(f.options)

	sort.Strings(keys)

	return strings.Join(keys, "|")
}
