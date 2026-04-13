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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"sigs.k8s.io/yaml"
)

const (
	// defaultCatalogBase points at the latest GitHub release of fluxcd/plugins.
	defaultCatalogBase = "https://github.com/fluxcd/plugins/releases/latest/download/"
	envCatalogBase     = "FLUXCD_PLUGIN_CATALOG"

	pluginAPIVersion = "cli.fluxcd.io/v1beta1"
	pluginKind       = "Plugin"
	catalogKind      = "PluginCatalog"
)

// PluginManifest represents a single plugin's manifest from the catalog.
type PluginManifest struct {
	// APIVersion is the manifest schema version (e.g. "cli.fluxcd.io/v1beta1").
	APIVersion string `json:"apiVersion"`

	// Kind is the manifest type, must be "Plugin".
	Kind string `json:"kind"`

	// Name is the plugin name used in "flux <name>" invocations.
	Name string `json:"name"`

	// Description is a short human-readable summary of the plugin.
	Description string `json:"description"`

	// Homepage is the URL to the plugin's documentation site.
	Homepage string `json:"homepage,omitempty"`

	// Source is the URL to the plugin's source repository.
	Source string `json:"source,omitempty"`

	// Bin is the binary name inside archives and the installed filename
	// (e.g. "flux-operator"). On Windows ".exe" is appended automatically.
	Bin string `json:"bin"`

	// Versions lists available versions, newest first.
	Versions []PluginVersion `json:"versions"`
}

// PluginVersion represents a version entry in a plugin manifest.
type PluginVersion struct {
	// Version is the semantic version string (e.g. "0.45.0").
	Version string `json:"version"`

	// Platforms lists the platform-specific binaries for this version.
	Platforms []PluginPlatform `json:"platforms"`
}

// PluginPlatform represents a platform-specific binary entry.
type PluginPlatform struct {
	// OS is the target operating system (e.g. "darwin", "linux", "windows").
	OS string `json:"os"`

	// Arch is the target architecture (e.g. "amd64", "arm64").
	Arch string `json:"arch"`

	// URL is the download URL for the archive or binary.
	URL string `json:"url"`

	// Checksum is the expected digest in "<algorithm>:<hex>" format
	// (e.g. "sha256:cd85d5d84d264...").
	Checksum string `json:"checksum"`

	// ExtractPath overrides the default binary lookup name inside archives.
	// When set, it is matched as an exact path within the archive (e.g.
	// "bin/flux-operator"). When empty, the archive is searched by the
	// base name derived from the manifest's Bin field.
	ExtractPath string `json:"extractPath,omitempty"`
}

// PluginCatalog represents the generated catalog.yaml file.
type PluginCatalog struct {
	// APIVersion is the catalog schema version (e.g. "cli.fluxcd.io/v1beta1").
	APIVersion string `json:"apiVersion"`

	// Kind is the catalog type, must be "PluginCatalog".
	Kind string `json:"kind"`

	// Plugins lists all available plugins in the catalog.
	Plugins []CatalogEntry `json:"plugins"`
}

// CatalogEntry is a single entry in the plugin catalog.
type CatalogEntry struct {
	// Name is the plugin name.
	Name string `json:"name"`

	// Description is a short human-readable summary of the plugin.
	Description string `json:"description"`

	// Homepage is the URL to the plugin's documentation site.
	Homepage string `json:"homepage,omitempty"`

	// Source is the URL to the plugin's source repository.
	Source string `json:"source,omitempty"`

	// License is the SPDX license identifier (e.g. "Apache-2.0").
	License string `json:"license,omitempty"`
}

// Receipt records what was installed for a plugin.
type Receipt struct {
	// Name is the plugin name (e.g. "operator").
	Name string `json:"name"`

	// Version is the installed semantic version.
	Version string `json:"version"`

	// InstalledAt is the RFC 3339 timestamp of the installation.
	InstalledAt string `json:"installedAt"`

	// Platform records the platform-specific details used for installation.
	Platform PluginPlatform `json:"platform"`
}

// CatalogClient fetches plugin manifests and catalogs from a remote URL.
type CatalogClient struct {
	// BaseURL is the catalog base URL for fetching manifests.
	BaseURL string

	// HTTPClient is the HTTP client used for catalog requests.
	HTTPClient *http.Client

	// GetEnv returns the value of an environment variable.
	GetEnv func(key string) string
}

// NewCatalogClient returns a CatalogClient with production defaults.
func NewCatalogClient() *CatalogClient {
	return &CatalogClient{
		BaseURL:    defaultCatalogBase,
		HTTPClient: newHTTPClient(30 * time.Second),
		GetEnv:     func(key string) string { return "" },
	}
}

// baseURL returns the effective catalog base URL.
func (c *CatalogClient) baseURL() string {
	if env := c.GetEnv(envCatalogBase); env != "" {
		return env
	}
	return c.BaseURL
}

// FetchManifest fetches a single plugin manifest from the catalog.
func (c *CatalogClient) FetchManifest(name string) (*PluginManifest, error) {
	url := c.baseURL() + name + ".yaml"
	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("plugin %q not found in catalog", name)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest for %q: %w", name, err)
	}

	if manifest.APIVersion != pluginAPIVersion {
		return nil, fmt.Errorf("plugin %q has unsupported apiVersion %q (expected %q)", name, manifest.APIVersion, pluginAPIVersion)
	}
	if manifest.Kind != pluginKind {
		return nil, fmt.Errorf("plugin %q has unexpected kind %q (expected %q)", name, manifest.Kind, pluginKind)
	}

	return &manifest, nil
}

// FetchCatalog fetches the generated catalog.yaml.
func (c *CatalogClient) FetchCatalog() (*PluginCatalog, error) {
	url := c.baseURL() + "catalog.yaml"
	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin catalog: %w", err)
	}

	var catalog PluginCatalog
	if err := yaml.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse plugin catalog: %w", err)
	}

	if catalog.APIVersion != pluginAPIVersion {
		return nil, fmt.Errorf("plugin catalog has unsupported apiVersion %q (expected %q)", catalog.APIVersion, pluginAPIVersion)
	}
	if catalog.Kind != catalogKind {
		return nil, fmt.Errorf("plugin catalog has unexpected kind %q (expected %q)", catalog.Kind, catalogKind)
	}

	return &catalog, nil
}

const maxResponseBytes = 10 << 20 // 10 MiB

func (c *CatalogClient) fetch(url string) ([]byte, error) {
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
}

// newHTTPClient returns a retrying HTTP client with the given timeout.
func newHTTPClient(timeout time.Duration) *http.Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.Logger = nil
	c := rc.StandardClient()
	c.Timeout = timeout
	return c
}

// ResolveVersion finds the requested version in the manifest.
// If version is empty, returns the first (latest) version.
func ResolveVersion(manifest *PluginManifest, version string) (*PluginVersion, error) {
	if len(manifest.Versions) == 0 {
		return nil, fmt.Errorf("plugin %q has no versions", manifest.Name)
	}

	if version == "" {
		return &manifest.Versions[0], nil
	}

	for i := range manifest.Versions {
		if manifest.Versions[i].Version == version {
			return &manifest.Versions[i], nil
		}
	}

	return nil, fmt.Errorf("version %q not found for plugin %q", version, manifest.Name)
}

// ResolvePlatform finds the platform entry matching the given OS and arch.
func ResolvePlatform(pv *PluginVersion, goos, goarch string) (*PluginPlatform, error) {
	for i := range pv.Platforms {
		if pv.Platforms[i].OS == goos && pv.Platforms[i].Arch == goarch {
			return &pv.Platforms[i], nil
		}
	}

	return nil, fmt.Errorf("no binary for %s/%s", goos, goarch)
}
