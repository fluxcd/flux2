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
	"testing"

	plugintypes "github.com/fluxcd/flux2/v2/pkg/plugin"
)

func TestFetchManifest(t *testing.T) {
	manifest := `
apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: operator
description: Flux Operator CLI
bin: flux-operator
versions:
  - version: 0.45.0
    platforms:
      - os: linux
        arch: amd64
        url: https://example.com/flux-operator_0.45.0_linux_amd64.tar.gz
        checksum: sha256:abc123
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/operator.yaml" {
			w.Write([]byte(manifest))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &CatalogClient{
		BaseURL:    server.URL + "/",
		HTTPClient: server.Client(),
		GetEnv:     func(key string) string { return "" },
	}

	m, err := client.FetchManifest("operator")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "operator" {
		t.Errorf("expected name 'operator', got %q", m.Name)
	}
	if m.Bin != "flux-operator" {
		t.Errorf("expected bin 'flux-operator', got %q", m.Bin)
	}
	if len(m.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(m.Versions))
	}
	if m.Versions[0].Version != "0.45.0" {
		t.Errorf("expected version '0.45.0', got %q", m.Versions[0].Version)
	}
}

func TestFetchManifestNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &CatalogClient{
		BaseURL:    server.URL + "/",
		HTTPClient: server.Client(),
		GetEnv:     func(key string) string { return "" },
	}

	_, err := client.FetchManifest("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchCatalog(t *testing.T) {
	catalog := `
apiVersion: cli.fluxcd.io/v1beta1
kind: PluginCatalog
plugins:
  - name: operator
    description: Flux Operator CLI
    homepage: https://fluxoperator.dev/
    source: https://github.com/controlplaneio-fluxcd/flux-operator
    license: AGPL-3.0
  - name: schema
    description: CRD schemas
    homepage: https://example.com/
    source: https://github.com/example/flux-schema
    license: Apache-2.0
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/catalog.yaml" {
			w.Write([]byte(catalog))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &CatalogClient{
		BaseURL:    server.URL + "/",
		HTTPClient: server.Client(),
		GetEnv:     func(key string) string { return "" },
	}

	c, err := client.FetchCatalog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(c.Plugins))
	}
	if c.Plugins[0].Name != "operator" {
		t.Errorf("expected name 'operator', got %q", c.Plugins[0].Name)
	}
	if c.Plugins[1].Name != "schema" {
		t.Errorf("expected name 'schema', got %q", c.Plugins[1].Name)
	}
}

func TestCatalogEnvOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom/catalog.yaml" {
			w.Write([]byte(`apiVersion: cli.fluxcd.io/v1beta1
kind: PluginCatalog
plugins: []
`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &CatalogClient{
		BaseURL:    "https://should-not-be-used/",
		HTTPClient: server.Client(),
		GetEnv: func(key string) string {
			if key == envCatalogBase {
				return server.URL + "/custom/"
			}
			return ""
		},
	}

	c, err := client.FetchCatalog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.Plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(c.Plugins))
	}
}

func TestResolveVersion(t *testing.T) {
	manifest := &plugintypes.Manifest{
		Name: "operator",
		Versions: []plugintypes.Version{
			{Version: "0.45.0"},
			{Version: "0.44.0"},
		},
	}

	t.Run("latest", func(t *testing.T) {
		v, err := ResolveVersion(manifest, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Version != "0.45.0" {
			t.Errorf("expected '0.45.0', got %q", v.Version)
		}
	})

	t.Run("specific", func(t *testing.T) {
		v, err := ResolveVersion(manifest, "0.44.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v.Version != "0.44.0" {
			t.Errorf("expected '0.44.0', got %q", v.Version)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ResolveVersion(manifest, "0.99.0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("no versions", func(t *testing.T) {
		_, err := ResolveVersion(&plugintypes.Manifest{Name: "empty"}, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestResolvePlatform(t *testing.T) {
	pv := &plugintypes.Version{
		Version: "0.45.0",
		Platforms: []plugintypes.Platform{
			{OS: "darwin", Arch: "arm64", URL: "https://example.com/darwin_arm64.tar.gz"},
			{OS: "linux", Arch: "amd64", URL: "https://example.com/linux_amd64.tar.gz"},
		},
	}

	t.Run("found", func(t *testing.T) {
		p, err := ResolvePlatform(pv, "darwin", "arm64")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.OS != "darwin" || p.Arch != "arm64" {
			t.Errorf("unexpected platform: %s/%s", p.OS, p.Arch)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ResolvePlatform(pv, "windows", "amd64")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
