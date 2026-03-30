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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckUpdateUpToDate(t *testing.T) {
	manifest := `
apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: operator
bin: flux-operator
versions:
  - version: 0.45.0
    platforms:
      - os: linux
        arch: amd64
        url: https://example.com/archive.tar.gz
        checksum: sha256:abc123
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(manifest))
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	// Write receipt with same version.
	receiptData := `name: operator
version: "0.45.0"
installedAt: "2026-03-28T20:05:00Z"
platform:
  os: linux
  arch: amd64
`
	os.WriteFile(filepath.Join(pluginDir, "flux-operator.yaml"), []byte(receiptData), 0o644)

	catalog := &CatalogClient{
		BaseURL:    server.URL + "/",
		HTTPClient: server.Client(),
		GetEnv:     func(key string) string { return "" },
	}

	result := CheckUpdate(pluginDir, "operator", catalog, "linux", "amd64")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !result.Skipped {
		t.Error("expected skipped=true")
	}
	if result.SkipReason != SkipReasonUpToDate {
		t.Errorf("expected %q, got %q", SkipReasonUpToDate, result.SkipReason)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	manifest := `
apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: operator
bin: flux-operator
versions:
  - version: 0.46.0
    platforms:
      - os: linux
        arch: amd64
        url: https://example.com/archive.tar.gz
        checksum: sha256:abc123
  - version: 0.45.0
    platforms:
      - os: linux
        arch: amd64
        url: https://example.com/archive.tar.gz
        checksum: sha256:def456
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(manifest))
	}))
	defer server.Close()

	pluginDir := t.TempDir()

	receiptData := `name: operator
version: "0.45.0"
installedAt: "2026-03-28T20:05:00Z"
platform:
  os: linux
  arch: amd64
`
	os.WriteFile(filepath.Join(pluginDir, "flux-operator.yaml"), []byte(receiptData), 0o644)

	catalog := &CatalogClient{
		BaseURL:    server.URL + "/",
		HTTPClient: server.Client(),
		GetEnv:     func(key string) string { return "" },
	}

	result := CheckUpdate(pluginDir, "operator", catalog, "linux", "amd64")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Skipped {
		t.Error("expected skipped=false")
	}
	if result.FromVersion != "0.45.0" {
		t.Errorf("expected from '0.45.0', got %q", result.FromVersion)
	}
	if result.ToVersion != "0.46.0" {
		t.Errorf("expected to '0.46.0', got %q", result.ToVersion)
	}
}

func TestCheckUpdateManualInstall(t *testing.T) {
	pluginDir := t.TempDir()

	// No receipt — manually installed.
	catalog := &CatalogClient{
		BaseURL:    "https://example.com/",
		HTTPClient: http.DefaultClient,
		GetEnv:     func(key string) string { return "" },
	}

	result := CheckUpdate(pluginDir, "operator", catalog, "linux", "amd64")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if !result.Skipped {
		t.Error("expected skipped=true")
	}
	if result.SkipReason != SkipReasonManual {
		t.Errorf("expected 'manually installed', got %q", result.SkipReason)
	}
}
