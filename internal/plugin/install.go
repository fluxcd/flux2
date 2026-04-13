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

	plugintypes "github.com/fluxcd/flux2/v2/pkg/plugin"
)

// Installer handles downloading, verifying, and installing plugins.
type Installer struct {
	// HTTPClient is the HTTP client used for downloading plugin archives.
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
func (inst *Installer) Install(pluginDir string, manifest *plugintypes.Manifest, pv *plugintypes.Version, plat *plugintypes.Platform) error {
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

	// manifest.Bin is the single source of truth for the installed binary
	// name (e.g. "flux-validate"). On Windows we always append ".exe".
	// For archives it's also the entry name we look up; for raw binaries
	// it's the rename target regardless of the URL's filename.
	binName := manifest.Bin
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	destPath := filepath.Join(pluginDir, binName)

	// extractTarget is the path to match inside the archive. When the
	// platform specifies an extractPath, use it verbatim (it may be a
	// nested path like "bin/flux-operator"). Otherwise fall back to
	// binName which matches by basename.
	extractTarget := binName
	if plat.ExtractPath != "" {
		extractTarget = plat.ExtractPath
	}

	format, err := detectArchiveFormat(tmpFile.Name(), plat.URL)
	if err != nil {
		return fmt.Errorf("failed to detect plugin format: %w", err)
	}

	switch format {
	case formatZip:
		err = extractFromZip(tmpFile.Name(), extractTarget, destPath)
	case formatTarGz:
		err = extractFromTarGz(tmpFile.Name(), extractTarget, destPath)
	case formatTar:
		err = extractFromTar(tmpFile.Name(), extractTarget, destPath)
	case formatBinary:
		err = copyPluginBinary(tmpFile.Name(), destPath)
	default:
		return fmt.Errorf("unexpected plugin format: %v", format)
	}
	if err != nil {
		return err
	}

	receipt := plugintypes.Receipt{
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
func ReadReceipt(pluginDir, name string) *plugintypes.Receipt {
	data, err := os.ReadFile(receiptPath(pluginDir, name))
	if err != nil {
		return nil
	}

	var receipt plugintypes.Receipt
	if err := yaml.Unmarshal(data, &receipt); err != nil {
		return nil
	}

	return &receipt
}

func receiptPath(pluginDir, name string) string {
	return filepath.Join(pluginDir, pluginPrefix+name+".yaml")
}

func writeReceipt(pluginDir, name string, receipt *plugintypes.Receipt) error {
	data, err := yaml.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("failed to marshal receipt: %w", err)
	}

	return os.WriteFile(receiptPath(pluginDir, name), data, 0o644)
}

// archiveFormat is the detected format of a downloaded plugin artifact.
type archiveFormat int

const (
	formatBinary archiveFormat = iota
	formatZip
	formatTarGz
	formatTar
)

// detectArchiveFormat determines the artifact format by first checking the URL
// extension, then falling back to magic-byte inspection. Returns formatBinary
// if neither indicates a known archive, in which case the downloaded file is
// installed as-is.
func detectArchiveFormat(path, url string) (archiveFormat, error) {
	switch lower := strings.ToLower(url); {
	case strings.HasSuffix(lower, ".zip"):
		return formatZip, nil
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return formatTarGz, nil
	case strings.HasSuffix(lower, ".tar"):
		return formatTar, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return formatBinary, err
	}
	defer f.Close()

	// Read enough bytes to cover the tar "ustar" magic at offset 257.
	var hdr [262]byte
	n, err := io.ReadFull(f, hdr[:])
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return formatBinary, err
	}

	// ZIP: PK\x03\x04 (file), PK\x05\x06 (empty), PK\x07\x08 (spanned).
	if n >= 4 && hdr[0] == 'P' && hdr[1] == 'K' &&
		(hdr[2] == 0x03 || hdr[2] == 0x05 || hdr[2] == 0x07) {
		return formatZip, nil
	}
	// gzip: \x1f\x8b
	if n >= 2 && hdr[0] == 0x1f && hdr[1] == 0x8b {
		return formatTarGz, nil
	}
	// tar: "ustar" at offset 257
	if n >= 262 && string(hdr[257:262]) == "ustar" {
		return formatTar, nil
	}

	return formatBinary, nil
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

	return extractTarStream(gr, targetName, destPath)
}

// extractFromTar extracts a named file from an uncompressed tar archive
// and streams it directly to destPath.
func extractFromTar(archivePath, targetName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return extractTarStream(f, targetName, destPath)
}

// matchArchiveEntry reports whether the archive entry name matches the
// target. If target contains a path separator it is matched as an exact
// path; otherwise only the base name of the entry is compared.
func matchArchiveEntry(entryName, target string) bool {
	if strings.Contains(target, "/") {
		return entryName == target
	}
	return filepath.Base(entryName) == target
}

// extractTarStream walks a tar stream and streams the first matching
// regular file to destPath. Non-regular entries (symlinks, devices,
// directories) and entries with unsafe paths are skipped.
func extractTarStream(r io.Reader, targetName, destPath string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if !filepath.IsLocal(header.Name) {
			continue
		}
		if !header.FileInfo().Mode().IsRegular() {
			continue
		}
		if matchArchiveEntry(header.Name, targetName) {
			return writeStreamToFile(tr, destPath)
		}
	}

	return fmt.Errorf("binary %q not found in archive", targetName)
}

// copyPluginBinary copies a raw downloaded binary to destPath with 0755 mode.
// Used when the downloaded artifact is not an archive.
func copyPluginBinary(srcPath, destPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded binary: %w", err)
	}
	defer src.Close()

	return writeStreamToFile(src, destPath)
}

// extractFromZip extracts a named file from a zip archive and streams it
// directly to destPath. Non-regular entries (symlinks, devices, directories)
// and entries with unsafe paths are skipped.
func extractFromZip(archivePath, targetName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !filepath.IsLocal(f.Name) {
			continue
		}
		if !f.FileInfo().Mode().IsRegular() {
			continue
		}
		if matchArchiveEntry(f.Name, targetName) {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open %q in zip: %w", targetName, err)
			}
			err = writeStreamToFile(rc, destPath)
			rc.Close()
			return err
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
		if closeErr := out.Close(); closeErr != nil {
			return fmt.Errorf("failed to write plugin binary: %w (also failed to close file: %v)", err, closeErr)
		}
		return fmt.Errorf("failed to write plugin binary: %w", err)
	}
	return out.Close()
}
