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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"bitbucket.org/creachadair/stringset"
	oci "github.com/fluxcd/pkg/oci/client"
	"github.com/fluxcd/pkg/tar"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/gonvenience/ytbx"
	"github.com/google/shlex"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
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
	differ      *differFlag
}

var diffArtifactArgs = newDiffArtifactArgs()

func newDiffArtifactArgs() diffArtifactFlags {
	return diffArtifactFlags{
		provider: flags.SourceOCIProvider(sourcev1.GenericOCIProvider),

		differ: &differFlag{
			options: map[string]differ{
				"dyff": dyffBuiltin{
					opts: []dyff.CompareOption{
						dyff.IgnoreOrderChanges(false),
						dyff.KubernetesEntityDetection(true),
					},
				},
				"external": externalDiff{},
				"unified":  unifiedDiff{},
			},
			description: map[string]string{
				"dyff":     `semantic diff for YAML inputs`,
				"external": `execute the command in the "` + externalDiffVar + `" environment variable`,
				"unified":  "generic unified diff for arbitrary text inputs",
			},
			value:  "unified",
			differ: unifiedDiff{},
		},
	}
}

func init() {
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.path, "path", "", "path to the directory or file containing the Kubernetes manifests (deprecated, use a second positional argument instead)")
	diffArtifactCmd.Flags().StringVar(&diffArtifactArgs.creds, "creds", "", "credentials for OCI registry in the format <username>[:<password>] if --provider is generic")
	diffArtifactCmd.Flags().Var(&diffArtifactArgs.provider, "provider", sourceOCIRepositoryArgs.provider.Description())
	diffArtifactCmd.Flags().StringSliceVar(&diffArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")
	diffArtifactCmd.Flags().BoolVarP(&diffArtifactArgs.brief, "brief", "q", false, "just print a line when the resources differ; does not output a list of changes")
	diffArtifactCmd.Flags().Var(diffArtifactArgs.differ, "differ", diffArtifactArgs.differ.usage())

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

func newMatcher(ignorePaths []string) gitignore.Matcher {
	var patterns []gitignore.Pattern

	for _, path := range ignorePaths {
		patterns = append(patterns, gitignore.ParsePattern(path, nil))
	}

	return gitignore.NewMatcher(patterns)
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

	return flags.differ.Diff(ctx, fromDir, toDir, flags.ignorePaths)
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
	defer archiveFH.Close()

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
	Diff(ctx context.Context, from, to string, ignorePaths []string) (string, error)
}

type unifiedDiff struct{}

func (d unifiedDiff) Diff(_ context.Context, fromDir, toDir string, ignorePaths []string) (string, error) {
	matcher := newMatcher(ignorePaths)

	fromFiles, err := filesInDir(fromDir, matcher)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "fromFiles = %v\n", fromFiles)

	toFiles, err := filesInDir(toDir, matcher)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "toFiles = %v\n", toFiles)

	allFiles := fromFiles.Union(toFiles)

	var sb strings.Builder

	for _, relPath := range allFiles.Elements() {
		diff, err := d.diffFiles(fromDir, toDir, relPath)
		if err != nil {
			return "", err
		}

		fmt.Fprint(&sb, diff)
	}

	return sb.String(), nil
}

func (d unifiedDiff) diffFiles(fromDir, toDir, relPath string) (string, error) {
	fromPath := filepath.Join(fromDir, relPath)
	fromData, err := d.readFile(fromPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("Only in %s: %s\n", toDir, relPath), nil
	case err != nil:
		return "", fmt.Errorf("readFile(%q): %w", fromPath, err)
	}

	toPath := filepath.Join(toDir, relPath)
	toData, err := d.readFile(toPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("Only in %s: %s\n", fromDir, relPath), nil
	case err != nil:
		return "", fmt.Errorf("readFile(%q): %w", toPath, err)
	}

	edits := myers.ComputeEdits(span.URIFromPath(fromPath), string(fromData), string(toData))
	return fmt.Sprint(gotextdiff.ToUnified(fromPath, toPath, string(fromData), edits)), nil
}

func (d unifiedDiff) readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

func splitPath(path string) []string {
	return strings.Split(path, string([]rune{filepath.Separator}))
}

func filesInDir(root string, matcher gitignore.Matcher) (stringset.Set, error) {
	var files stringset.Set

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("filepath.Rel(%q, %q): %w", root, path, err)
		}

		if matcher.Match(splitPath(relPath), d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		files.Add(relPath)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, err
}

// externalDiff implements the differ interface using an external diff command.
type externalDiff struct{}

// externalDiffVar is the environment variable users can use to overwrite the external diff command.
const externalDiffVar = "FLUX_EXTERNAL_DIFF"

func (externalDiff) Diff(ctx context.Context, fromDir, toDir string, ignorePaths []string) (string, error) {
	cmdline := os.Getenv(externalDiffVar)
	if cmdline == "" {
		return "", fmt.Errorf("the required %q environment variable is unset", externalDiffVar)
	}

	args, err := shlex.Split(cmdline)
	if err != nil {
		return "", fmt.Errorf("shlex.Split(%q): %w", cmdline, err)
	}

	var executable string
	executable, args = args[0], args[1:]

	for _, path := range ignorePaths {
		args = append(args, "--exclude", path)
	}

	args = append(args, fromDir, toDir)

	cmd := exec.CommandContext(ctx, executable, args...)

	var stdout bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// exit code 1 only means there was a difference => ignore
	} else if err != nil {
		return "", fmt.Errorf("executing %q: %w", executable, err)
	}

	return stdout.String(), nil
}

// dyffBuiltin implements the differ interface using `dyff`, a semantic diff for YAML documents.
type dyffBuiltin struct {
	opts []dyff.CompareOption
}

func (d dyffBuiltin) Diff(ctx context.Context, fromDir, toDir string, _ []string) (string, error) {
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

// differFlag implements pflag.Value for choosing a diffing implementation.
type differFlag struct {
	options     map[string]differ
	description map[string]string
	value       string
	differ
}

func (f *differFlag) Set(s string) error {
	d, ok := f.options[s]
	if !ok {
		return fmt.Errorf("invalid value: %q", s)
	}

	f.value = s
	f.differ = d

	return nil
}

func (f *differFlag) String() string {
	return f.value
}

func (f *differFlag) Type() string {
	keys := maps.Keys(f.options)

	sort.Strings(keys)

	return strings.Join(keys, "|")
}

func (f *differFlag) usage() string {
	var b strings.Builder
	fmt.Fprint(&b, "how the diff is generated:")

	keys := maps.Keys(f.options)

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Fprintf(&b, "\n  %q: %s", key, f.description[key])
	}

	return b.String()
}
