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

// Package plugin defines the public types for the Flux CLI plugin system.
// These types represent the plugin catalog schema (cli.fluxcd.io/v1beta1)
// and are safe for use by external consumers.
package plugin

const (
	// APIVersion is the plugin manifest schema version.
	APIVersion = "cli.fluxcd.io/v1beta1"

	// PluginKind is the kind for plugin manifests.
	PluginKind = "Plugin"

	// CatalogKind is the kind for the plugin catalog.
	CatalogKind = "PluginCatalog"
)

// Manifest represents a single plugin's manifest from the catalog.
type Manifest struct {
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
	Versions []Version `json:"versions"`
}

// Version represents a version entry in a plugin manifest.
type Version struct {
	// Version is the semantic version string (e.g. "0.45.0").
	Version string `json:"version"`

	// Platforms lists the platform-specific binaries for this version.
	Platforms []Platform `json:"platforms"`
}

// Platform represents a platform-specific binary entry.
type Platform struct {
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

// Catalog represents the generated catalog.yaml file.
type Catalog struct {
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
	Homepage string `json:"homepage"`

	// Source is the URL to the plugin's source repository.
	Source string `json:"source"`

	// License is the SPDX license identifier (e.g. "Apache-2.0").
	License string `json:"license"`
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
	Platform Platform `json:"platform"`
}
