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

	plugintypes "github.com/fluxcd/flux2/v2/pkg/plugin"
)

const (
	// defaultCatalogBase points at the latest GitHub release of fluxcd/plugins.
	defaultCatalogBase = "https://github.com/fluxcd/plugins/releases/latest/download/"
	envCatalogBase     = "FLUXCD_PLUGIN_CATALOG"
)

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
func (c *CatalogClient) FetchManifest(name string) (*plugintypes.Manifest, error) {
	url := c.baseURL() + name + ".yaml"
	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("plugin %q not found in catalog", name)
	}

	var manifest plugintypes.Manifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest for %q: %w", name, err)
	}

	if manifest.APIVersion != plugintypes.APIVersion {
		return nil, fmt.Errorf("plugin %q has unsupported apiVersion %q (expected %q)", name, manifest.APIVersion, plugintypes.APIVersion)
	}
	if manifest.Kind != plugintypes.PluginKind {
		return nil, fmt.Errorf("plugin %q has unexpected kind %q (expected %q)", name, manifest.Kind, plugintypes.PluginKind)
	}

	return &manifest, nil
}

// FetchCatalog fetches the generated catalog.yaml.
func (c *CatalogClient) FetchCatalog() (*plugintypes.Catalog, error) {
	url := c.baseURL() + "catalog.yaml"
	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin catalog: %w", err)
	}

	var catalog plugintypes.Catalog
	if err := yaml.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse plugin catalog: %w", err)
	}

	if catalog.APIVersion != plugintypes.APIVersion {
		return nil, fmt.Errorf("plugin catalog has unsupported apiVersion %q (expected %q)", catalog.APIVersion, plugintypes.APIVersion)
	}
	if catalog.Kind != plugintypes.CatalogKind {
		return nil, fmt.Errorf("plugin catalog has unexpected kind %q (expected %q)", catalog.Kind, plugintypes.CatalogKind)
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
func ResolveVersion(manifest *plugintypes.Manifest, version string) (*plugintypes.Version, error) {
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
func ResolvePlatform(pv *plugintypes.Version, goos, goarch string) (*plugintypes.Platform, error) {
	for i := range pv.Platforms {
		if pv.Platforms[i].OS == goos && pv.Platforms[i].Arch == goarch {
			return &pv.Platforms[i], nil
		}
	}

	return nil, fmt.Errorf("no binary for %s/%s", goos, goarch)
}

// DigestMatch holds the version and platform resolved from a digest lookup.
type DigestMatch struct {
	Version  *plugintypes.Version
	Platform *plugintypes.Platform
}

// ResolveByDigest scans all versions and platforms for a checksum matching
// digest. The digest must be in "algorithm:hex" format (e.g.
// "sha256:06e0a38..."). Only platforms matching goos/goarch are considered.
// Returns the first match (versions are ordered newest-first in the manifest).
func ResolveByDigest(manifest *plugintypes.Manifest, digest, goos, goarch string) (*DigestMatch, error) {
	if len(manifest.Versions) == 0 {
		return nil, fmt.Errorf("plugin %q has no versions", manifest.Name)
	}

	for i := range manifest.Versions {
		for j := range manifest.Versions[i].Platforms {
			p := &manifest.Versions[i].Platforms[j]
			if p.OS == goos && p.Arch == goarch && p.Checksum == digest {
				return &DigestMatch{
					Version:  &manifest.Versions[i],
					Platform: p,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("digest %q not found for plugin %q on %s/%s", digest, manifest.Name, goos, goarch)
}
