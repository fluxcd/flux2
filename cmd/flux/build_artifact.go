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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fluxcd/pkg/oci"
	"github.com/fluxcd/pkg/sourceignore"
)

var buildArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Build artifact",
	Long: `The build artifact command creates a tgz file with the manifests
from the given directory or a single manifest file.`,
	Example: `  # Build the given manifests directory into an artifact
  flux build artifact --path ./path/to/local/manifests --output ./path/to/artifact.tgz

  # Build the given single manifest file into an artifact
  flux build artifact --path ./path/to/local/manifest.yaml --output ./path/to/artifact.tgz

  # List the files bundled in the artifact
  tar -ztvf ./path/to/artifact.tgz
`,
	RunE: buildArtifactCmdRun,
}

type buildArtifactFlags struct {
	output          string
	path            string
	ignorePaths     []string
	resolveSymlinks bool
}

var excludeOCI = append(strings.Split(sourceignore.ExcludeVCS, ","), strings.Split(sourceignore.ExcludeExt, ",")...)

var buildArtifactArgs buildArtifactFlags

func init() {
	buildArtifactCmd.Flags().StringVarP(&buildArtifactArgs.path, "path", "p", "", "Path to the directory where the Kubernetes manifests are located.")
	buildArtifactCmd.Flags().StringVarP(&buildArtifactArgs.output, "output", "o", "artifact.tgz", "Path to where the artifact tgz file should be written.")
	buildArtifactCmd.Flags().StringSliceVar(&buildArtifactArgs.ignorePaths, "ignore-paths", excludeOCI, "set paths to ignore in .gitignore format")
	buildArtifactCmd.Flags().BoolVar(&buildArtifactArgs.resolveSymlinks, "resolve-symlinks", false, "resolve symlinks by copying their targets into the artifact")

	buildCmd.AddCommand(buildArtifactCmd)
}

func buildArtifactCmdRun(cmd *cobra.Command, args []string) error {
	if buildArtifactArgs.path == "" {
		return fmt.Errorf("invalid path %q", buildArtifactArgs.path)
	}

	path := buildArtifactArgs.path
	var err error
	if buildArtifactArgs.path == "-" {
		path, err = saveReaderToFile(os.Stdin)
		if err != nil {
			return err
		}

		defer os.Remove(path)
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("invalid path '%s', must point to an existing directory or file", path)
	}

	if buildArtifactArgs.resolveSymlinks {
		resolved, cleanupDir, err := resolveSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolving symlinks failed: %w", err)
		}
		defer os.RemoveAll(cleanupDir)
		path = resolved
	}

	logger.Actionf("building artifact from %s", path)

	ociClient := oci.NewClient(oci.DefaultOptions())
	if err := ociClient.Build(buildArtifactArgs.output, path, buildArtifactArgs.ignorePaths); err != nil {
		return fmt.Errorf("building artifact failed, error: %w", err)
	}

	logger.Successf("artifact created at %s", buildArtifactArgs.output)
	return nil
}

// resolveSymlinks creates a temporary directory with symlinks resolved to their
// real file contents. This allows building artifacts from symlink trees (e.g.,
// those created by Nix) where the actual files live outside the source directory.
// It returns the resolved path and the temporary directory path for cleanup.
func resolveSymlinks(srcPath string) (string, string, error) {
	absPath, err := filepath.Abs(srcPath)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", err
	}

	// For a single file, resolve the symlink and return the path to the
	// copied file within the temp dir, preserving file semantics for callers.
	if !info.IsDir() {
		resolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", "", fmt.Errorf("resolving symlink for %s: %w", absPath, err)
		}
		tmpDir, err := os.MkdirTemp("", "flux-artifact-*")
		if err != nil {
			return "", "", err
		}
		dst := filepath.Join(tmpDir, filepath.Base(absPath))
		if err := copyFile(resolved, dst); err != nil {
			os.RemoveAll(tmpDir)
			return "", "", err
		}
		return dst, tmpDir, nil
	}

	tmpDir, err := os.MkdirTemp("", "flux-artifact-*")
	if err != nil {
		return "", "", err
	}

	visited := make(map[string]bool)
	if err := copyDir(absPath, tmpDir, visited); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	return tmpDir, tmpDir, nil
}

// copyDir recursively copies the contents of srcDir to dstDir, resolving any
// symlinks encountered along the way. The visited map tracks resolved real
// directory paths to detect and break symlink cycles.
func copyDir(srcDir, dstDir string, visited map[string]bool) error {
	real, err := filepath.EvalSymlinks(srcDir)
	if err != nil {
		return fmt.Errorf("resolving symlink %s: %w", srcDir, err)
	}
	abs, err := filepath.Abs(real)
	if err != nil {
		return fmt.Errorf("getting absolute path for %s: %w", real, err)
	}
	if visited[abs] {
		return nil // break the cycle
	}
	visited[abs] = true

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		// Resolve symlinks to get the real path and info.
		realPath, err := filepath.EvalSymlinks(srcPath)
		if err != nil {
			return fmt.Errorf("resolving symlink %s: %w", srcPath, err)
		}
		realInfo, err := os.Stat(realPath)
		if err != nil {
			return fmt.Errorf("stat resolved path %s: %w", realPath, err)
		}

		if realInfo.IsDir() {
			if err := os.MkdirAll(dstPath, realInfo.Mode()); err != nil {
				return err
			}
			// Recursively copy the resolved directory contents.
			if err := copyDir(realPath, dstPath, visited); err != nil {
				return err
			}
			continue
		}

		if !realInfo.Mode().IsRegular() {
			continue
		}

		if err := copyFile(realPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func saveReaderToFile(reader io.Reader) (string, error) {
	b, err := io.ReadAll(bufio.NewReader(reader))
	if err != nil {
		return "", err
	}
	b = bytes.TrimRight(b, "\r\n")
	f, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		return "", fmt.Errorf("unable to create temp dir for stdin")
	}

	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return "", fmt.Errorf("error writing stdin to file: %w", err)
	}

	return f.Name(), nil
}
