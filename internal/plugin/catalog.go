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
	APIVersion  string          `json:"apiVersion"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Homepage    string          `json:"homepage,omitempty"`
	Source      string          `json:"source,omitempty"`
	Bin         string          `json:"bin"`
	Versions    []PluginVersion `json:"versions"`
}

// PluginVersion represents a version entry in a plugin manifest.
type PluginVersion struct {
	Version   string           `json:"version"`
	Platforms []PluginPlatform `json:"platforms"`
}

// PluginPlatform represents a platform-specific binary entry.
type PluginPlatform struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

// PluginCatalog represents the generated catalog.yaml file.
type PluginCatalog struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Plugins    []CatalogEntry `json:"plugins"`
}

// CatalogEntry is a single entry in the plugin catalog.
type CatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage,omitempty"`
	Source      string `json:"source,omitempty"`
	License     string `json:"license,omitempty"`
}

// Receipt records what was installed for a plugin.
type Receipt struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	InstalledAt string         `json:"installedAt"`
	Platform    PluginPlatform `json:"platform"`
}

// CatalogClient fetches plugin manifests and catalogs from a remote URL.
type CatalogClient struct {
	BaseURL    string
	HTTPClient *http.Client
	GetEnv     func(key string) string
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
