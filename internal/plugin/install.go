/*
Copyright 2026 The Flux authors

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

package plugin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

// Installer handles downloading, verifying, and installing plugins.
type Installer struct {
	HTTPClient *http.Client
}

// NewInstaller returns an Installer with production defaults.
func NewInstaller() *Installer {
	return &Installer{
		HTTPClient: newHTTPClient(5 * time.Minute),
	}
}

// Install downloads, verifies, extracts, and installs a plugin binary
// to the given plugin directory.
func (inst *Installer) Install(pluginDir string, manifest *PluginManifest, pv *PluginVersion, plat *PluginPlatform) error {
	tmpFile, err := os.CreateTemp("", "flux-plugin-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := inst.HTTPClient.Get(plat.URL)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download plugin: HTTP %d", resp.StatusCode)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	tmpFile.Close()

	actualChecksum := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if actualChecksum != plat.Checksum {
		return fmt.Errorf("checksum verification failed (expected: %s, got: %s)", plat.Checksum, actualChecksum)
	}

	binName := manifest.Bin
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	destName := pluginPrefix + manifest.Name
	if runtime.GOOS == "windows" {
		destName += ".exe"
	}
	destPath := filepath.Join(pluginDir, destName)

	if strings.HasSuffix(plat.URL, ".zip") {
		err = extractFromZip(tmpFile.Name(), binName, destPath)
	} else {
		err = extractFromTarGz(tmpFile.Name(), binName, destPath)
	}
	if err != nil {
		return err
	}

	receipt := Receipt{
		Name:        manifest.Name,
		Version:     pv.Version,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Platform:    *plat,
	}
	return writeReceipt(pluginDir, manifest.Name, &receipt)
}

// Uninstall removes a plugin binary (or symlink) and its receipt from the
// plugin directory. Returns an error if the plugin is not installed.
func Uninstall(pluginDir, name string) error {
	binName := pluginPrefix + name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	binPath := filepath.Join(pluginDir, binName)

	// Use Lstat so we detect symlinks without following them.
	if _, err := os.Lstat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	if err := os.Remove(binPath); err != nil {
		return fmt.Errorf("failed to remove plugin binary: %w", err)
	}

	// Receipt is optional (manually installed plugins don't have one).
	if err := os.Remove(receiptPath(pluginDir, name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plugin receipt: %w", err)
	}

	return nil
}

// ReadReceipt reads the install receipt for a plugin.
// Returns nil if no receipt exists.
func ReadReceipt(pluginDir, name string) *Receipt {
	data, err := os.ReadFile(receiptPath(pluginDir, name))
	if err != nil {
		return nil
	}

	var receipt Receipt
	if err := yaml.Unmarshal(data, &receipt); err != nil {
		return nil
	}

	return &receipt
}

func receiptPath(pluginDir, name string) string {
	return filepath.Join(pluginDir, pluginPrefix+name+".yaml")
}

func writeReceipt(pluginDir, name string, receipt *Receipt) error {
	data, err := yaml.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("failed to marshal receipt: %w", err)
	}

	return os.WriteFile(receiptPath(pluginDir, name), data, 0o644)
}

// extractFromTarGz extracts a named file from a tar.gz archive
// and streams it directly to destPath.
func extractFromTarGz(archivePath, targetName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if filepath.IsAbs(header.Name) || strings.Contains(header.Name, "..") {
			continue
		}
		if filepath.Base(header.Name) == targetName && header.Typeflag == tar.TypeReg {
			return writeStreamToFile(tr, destPath)
		}
	}

	return fmt.Errorf("binary %q not found in archive", targetName)
}

// extractFromZip extracts a named file from a zip archive
// and streams it directly to destPath.
func extractFromZip(archivePath, targetName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == targetName && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open %q in zip: %w", targetName, err)
			}
			defer rc.Close()
			return writeStreamToFile(rc, destPath)
		}
	}

	return fmt.Errorf("binary %q not found in archive", targetName)
}

func writeStreamToFile(r io.Reader, destPath string) error {
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destPath, err)
	}

	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return fmt.Errorf("failed to write plugin binary: %w", err)
	}
	return out.Close()
}
