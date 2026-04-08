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
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// createTestTarGz creates a tar.gz archive containing a single file.
func createTestTarGz(name string, content []byte) ([]byte, error) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}

	tw.Close()
	gw.Close()
	return buf.Bytes(), nil
}

// createTestTar creates an uncompressed tar archive containing a single file.
func createTestTar(name string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	hdr := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}

	tw.Close()
	return buf.Bytes(), nil
}

// tarEntry describes a single entry for createTestTarGzMulti.
type tarEntry struct {
	header  tar.Header
	content []byte
}

// createTestTarGzMulti creates a tar.gz archive with arbitrary entries.
// Used to test rejection of unsafe or non-regular entries.
func createTestTarGzMulti(entries []tarEntry) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		hdr := e.header
		hdr.Size = int64(len(e.content))
		if err := tw.WriteHeader(&hdr); err != nil {
			return nil, err
		}
		if len(e.content) > 0 {
			if _, err := tw.Write(e.content); err != nil {
				return nil, err
			}
		}
	}

	tw.Close()
	gw.Close()
	return buf.Bytes(), nil
}

// zipEntry describes a single entry for createTestZip.
type zipEntry struct {
	name    string
	mode    fs.FileMode
	content []byte
}

// createTestZip creates a zip archive with arbitrary entries. Entries may
// carry Unix mode bits (e.g. os.ModeSymlink) to exercise non-regular files.
func createTestZip(entries []zipEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, e := range entries {
		hdr := &zip.FileHeader{
			Name:   e.name,
			Method: zip.Deflate,
		}
		mode := e.mode
		if mode == 0 {
			mode = 0o755
		}
		hdr.SetMode(mode)

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(e.content); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestInstall(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho hello")
	archive, err := createTestTarGz("flux-operator", binaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(archive))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	manifest := &PluginManifest{
		Name: "operator",
		Bin:  "flux-operator",
	}
	pv := &PluginVersion{Version: "0.45.0"}
	plat := &PluginPlatform{
		OS:       "linux",
		Arch:     "amd64",
		URL:      server.URL + "/flux-operator_0.45.0_linux_amd64.tar.gz",
		Checksum: checksum,
	}

	installer := &Installer{HTTPClient: server.Client()}
	if err := installer.Install(pluginDir, manifest, pv, plat); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify binary was written.
	binPath := filepath.Join(pluginDir, "flux-operator")
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("binary not found: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Errorf("binary content mismatch")
	}

	// Verify receipt was written.
	receipt := ReadReceipt(pluginDir, "operator")
	if receipt == nil {
		t.Fatal("receipt not found")
	}
	if receipt.Version != "0.45.0" {
		t.Errorf("expected version '0.45.0', got %q", receipt.Version)
	}
	if receipt.Name != "operator" {
		t.Errorf("expected name 'operator', got %q", receipt.Name)
	}
}

func TestInstallChecksumMismatch(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho hello")
	archive, err := createTestTarGz("flux-operator", binaryContent)
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	manifest := &PluginManifest{Name: "operator", Bin: "flux-operator"}
	pv := &PluginVersion{Version: "0.45.0"}
	plat := &PluginPlatform{
		OS:       "linux",
		Arch:     "amd64",
		URL:      server.URL + "/archive.tar.gz",
		Checksum: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}

	installer := &Installer{HTTPClient: server.Client()}
	err = installer.Install(pluginDir, manifest, pv, plat)
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("checksum verification failed")) {
		t.Errorf("expected checksum error, got: %v", err)
	}
}

func TestInstallBinaryNotInArchive(t *testing.T) {
	// Archive contains "wrong-name" instead of "flux-operator".
	archive, err := createTestTarGz("wrong-name", []byte("content"))
	if err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(archive))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	manifest := &PluginManifest{Name: "operator", Bin: "flux-operator"}
	pv := &PluginVersion{Version: "0.45.0"}
	plat := &PluginPlatform{
		OS:       "linux",
		Arch:     "amd64",
		URL:      server.URL + "/archive.tar.gz",
		Checksum: checksum,
	}

	installer := &Installer{HTTPClient: server.Client()}
	err = installer.Install(pluginDir, manifest, pv, plat)
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("not found in archive")) {
		t.Errorf("expected 'not found in archive' error, got: %v", err)
	}
}

func TestUninstall(t *testing.T) {
	pluginDir := t.TempDir()

	// Create fake binary and receipt.
	binPath := filepath.Join(pluginDir, "flux-testplugin")
	os.WriteFile(binPath, []byte("binary"), 0o755)
	receiptPath := filepath.Join(pluginDir, "flux-testplugin.yaml")
	os.WriteFile(receiptPath, []byte("name: testplugin"), 0o644)

	if err := Uninstall(pluginDir, "testplugin"); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Error("binary was not removed")
	}
	if _, err := os.Stat(receiptPath); !os.IsNotExist(err) {
		t.Error("receipt was not removed")
	}
}

func TestUninstallNonExistent(t *testing.T) {
	pluginDir := t.TempDir()

	err := Uninstall(pluginDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent plugin, got nil")
	}
	if !strings.Contains(err.Error(), "is not installed") {
		t.Errorf("expected 'is not installed' error, got: %v", err)
	}
}

func TestUninstallSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	pluginDir := t.TempDir()

	// Create a real binary and symlink it into the plugin dir.
	realBin := filepath.Join(t.TempDir(), "flux-operator")
	os.WriteFile(realBin, []byte("real binary"), 0o755)

	linkPath := filepath.Join(pluginDir, "flux-linked")
	os.Symlink(realBin, linkPath)

	if err := Uninstall(pluginDir, "linked"); err != nil {
		t.Fatalf("uninstall symlink failed: %v", err)
	}

	// Symlink should be removed.
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Error("symlink was not removed")
	}
	// Original binary should still exist.
	if _, err := os.Stat(realBin); err != nil {
		t.Error("original binary was removed — symlink removal should not affect target")
	}
}

func TestUninstallManualBinary(t *testing.T) {
	pluginDir := t.TempDir()

	// Manually copied binary with no receipt.
	binPath := filepath.Join(pluginDir, "flux-manual")
	os.WriteFile(binPath, []byte("binary"), 0o755)

	if err := Uninstall(pluginDir, "manual"); err != nil {
		t.Fatalf("uninstall manual binary failed: %v", err)
	}

	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Error("binary was not removed")
	}
}

func TestReadReceipt(t *testing.T) {
	pluginDir := t.TempDir()

	t.Run("exists", func(t *testing.T) {
		receiptData := `name: operator
version: "0.45.0"
installedAt: "2026-03-28T20:05:00Z"
platform:
  os: darwin
  arch: arm64
  url: https://example.com/archive.tar.gz
  checksum: sha256:abc123
`
		os.WriteFile(filepath.Join(pluginDir, "flux-operator.yaml"), []byte(receiptData), 0o644)

		receipt := ReadReceipt(pluginDir, "operator")
		if receipt == nil {
			t.Fatal("expected receipt, got nil")
		}
		if receipt.Version != "0.45.0" {
			t.Errorf("expected version '0.45.0', got %q", receipt.Version)
		}
		if receipt.Platform.OS != "darwin" {
			t.Errorf("expected OS 'darwin', got %q", receipt.Platform.OS)
		}
	})

	t.Run("not exists", func(t *testing.T) {
		receipt := ReadReceipt(pluginDir, "nonexistent")
		if receipt != nil {
			t.Error("expected nil receipt")
		}
	})
}

func TestInstallRawBinary(t *testing.T) {
	// Bytes that don't match zip/gzip/tar magic — treated as a raw binary.
	binaryContent := []byte("#!/bin/sh\necho raw plugin")
	checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(binaryContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(binaryContent)
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	manifest := &PluginManifest{
		Name: "validate",
		Bin:  "flux-validate",
	}
	pv := &PluginVersion{Version: "1.2.3"}
	plat := &PluginPlatform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		// URL filename deliberately differs from manifest.Bin — mimics a
		// typical GitHub release asset that includes platform/version in
		// the name. The installer must rename to manifest.Bin.
		URL:      server.URL + "/download/flux-validate-" + runtime.GOARCH + "-v1.2.3",
		Checksum: checksum,
	}

	installer := &Installer{HTTPClient: server.Client()}
	if err := installer.Install(pluginDir, manifest, pv, plat); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// The installed file must be named exactly manifest.Bin (+ .exe on Windows),
	// regardless of what the URL path looked like.
	wantName := "flux-validate"
	if runtime.GOOS == "windows" {
		wantName += ".exe"
	}
	binPath := filepath.Join(pluginDir, wantName)
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("binary not found at %s: %v", binPath, err)
	}
	if !bytes.Equal(data, binaryContent) {
		t.Errorf("binary content mismatch: got %q, want %q", data, binaryContent)
	}

	// Nothing should have been written under the URL-derived name.
	urlDerived := filepath.Join(pluginDir, "flux-validate-"+runtime.GOARCH+"-v1.2.3")
	if _, err := os.Stat(urlDerived); !os.IsNotExist(err) {
		t.Errorf("unexpected file at URL-derived path %s", urlDerived)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(binPath)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("binary is not executable: mode %v", info.Mode())
		}
	}

	// Raw binary install must still produce a receipt.
	if receipt := ReadReceipt(pluginDir, "validate"); receipt == nil {
		t.Fatal("receipt not found")
	}
}

func TestDetectArchiveFormat(t *testing.T) {
	tarGz, err := createTestTarGz("bin", []byte("content"))
	if err != nil {
		t.Fatalf("createTestTarGz: %v", err)
	}
	plainTar, err := createTestTar("bin", []byte("content"))
	if err != nil {
		t.Fatalf("createTestTar: %v", err)
	}

	tests := []struct {
		name    string
		url     string
		content []byte
		want    archiveFormat
	}{
		// Extension-based detection takes precedence over content.
		{"zip extension", "https://example.com/plugin.zip", []byte("ignored"), formatZip},
		{"tar.gz extension", "https://example.com/plugin.tar.gz", []byte("ignored"), formatTarGz},
		{"tgz extension", "https://example.com/plugin.tgz", []byte("ignored"), formatTarGz},
		{"tar extension", "https://example.com/plugin.tar", []byte("ignored"), formatTar},
		{"uppercase extension", "https://example.com/PLUGIN.ZIP", []byte("ignored"), formatZip},

		// Magic-byte detection when extension is absent or unrecognized.
		{"zip magic no extension", "https://example.com/download", []byte{'P', 'K', 0x03, 0x04, 0, 0, 0, 0}, formatZip},
		{"gzip magic no extension", "https://example.com/download", tarGz, formatTarGz},
		{"tar magic no extension", "https://example.com/download", plainTar, formatTar},

		// Fallback to raw binary.
		{"unknown content", "https://example.com/download", []byte("#!/bin/sh\necho hi"), formatBinary},
		{"short file", "https://example.com/download", []byte("ab"), formatBinary},
		{"empty file", "https://example.com/download", []byte{}, formatBinary},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := filepath.Join(t.TempDir(), "artifact")
			if err := os.WriteFile(tmp, tc.content, 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := detectArchiveFormat(tmp, tc.url)
			if err != nil {
				t.Fatalf("detectArchiveFormat: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtractFromTarGz(t *testing.T) {
	content := []byte("test binary content")
	archive, err := createTestTarGz("flux-operator", content)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
	os.WriteFile(tmpFile, archive, 0o644)

	destPath := filepath.Join(t.TempDir(), "flux-operator")
	if err := extractFromTarGz(tmpFile, "flux-operator", destPath); err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestExtractFromTarGzRejectsUnsafeEntries(t *testing.T) {
	content := []byte("legit content")

	// Archive contains, in order:
	//  1. A symlink whose basename matches the target (must be skipped).
	//  2. A regular entry with ".." in the path (must be skipped).
	//  3. An absolute-path entry (must be skipped).
	//  4. A legitimate regular file that must be extracted.
	entries := []tarEntry{
		{
			header: tar.Header{
				Name:     "flux-operator",
				Typeflag: tar.TypeSymlink,
				Linkname: "/etc/passwd",
				Mode:     0o777,
			},
		},
		{
			header: tar.Header{
				Name:     "../flux-operator",
				Typeflag: tar.TypeReg,
				Mode:     0o755,
			},
			content: []byte("malicious"),
		},
		{
			header: tar.Header{
				Name:     "/absolute/flux-operator",
				Typeflag: tar.TypeReg,
				Mode:     0o755,
			},
			content: []byte("malicious"),
		},
		{
			header: tar.Header{
				Name:     "bin/flux-operator",
				Typeflag: tar.TypeReg,
				Mode:     0o755,
			},
			content: content,
		},
	}

	archive, err := createTestTarGzMulti(entries)
	if err != nil {
		t.Fatalf("createTestTarGzMulti: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
	os.WriteFile(tmpFile, archive, 0o644)

	destPath := filepath.Join(t.TempDir(), "flux-operator")
	if err := extractFromTarGz(tmpFile, "flux-operator", destPath); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("extracted content mismatch: got %q, want %q", data, content)
	}
}

func TestExtractFromZip(t *testing.T) {
	content := []byte("test binary content")
	archive, err := createTestZip([]zipEntry{
		{name: "flux-operator", content: content},
	})
	if err != nil {
		t.Fatalf("createTestZip: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.zip")
	os.WriteFile(tmpFile, archive, 0o644)

	destPath := filepath.Join(t.TempDir(), "flux-operator")
	if err := extractFromZip(tmpFile, "flux-operator", destPath); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

func TestExtractFromZipRejectsUnsafeEntries(t *testing.T) {
	content := []byte("legit content")

	// Archive contains, in order:
	//  1. A symlink whose basename matches the target (must be skipped).
	//  2. An entry with ".." in the path (must be skipped).
	//  3. An absolute-path entry (must be skipped).
	//  4. A directory entry whose basename matches (must be skipped).
	//  5. A legitimate regular file that must be extracted.
	entries := []zipEntry{
		{
			name:    "flux-operator",
			mode:    fs.ModeSymlink | 0o777,
			content: []byte("/etc/passwd"),
		},
		{
			name:    "../flux-operator",
			content: []byte("malicious"),
		},
		{
			name:    "/absolute/flux-operator",
			content: []byte("malicious"),
		},
		{
			name: "flux-operator/",
			mode: fs.ModeDir | 0o755,
		},
		{
			name:    "bin/flux-operator",
			content: content,
		},
	}

	archive, err := createTestZip(entries)
	if err != nil {
		t.Fatalf("createTestZip: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.zip")
	os.WriteFile(tmpFile, archive, 0o644)

	destPath := filepath.Join(t.TempDir(), "flux-operator")
	if err := extractFromZip(tmpFile, "flux-operator", destPath); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("extracted content mismatch: got %q, want %q", data, content)
	}
}

func TestExtractFromTarGzNotFound(t *testing.T) {
	archive, err := createTestTarGz("other-binary", []byte("content"))
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
	os.WriteFile(tmpFile, archive, 0o644)

	destPath := filepath.Join(t.TempDir(), "flux-operator")
	err = extractFromTarGz(tmpFile, "flux-operator", destPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
