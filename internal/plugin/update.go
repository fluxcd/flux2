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

const (
	SkipReasonManual   = "manually installed"
	SkipReasonUpToDate = "already up to date"
)

// UpdateResult represents the outcome of updating a single plugin.
// When an update is available, Manifest, Version and Platform are
// populated so the caller can install without re-fetching or re-resolving.
type UpdateResult struct {
	// Name is the plugin name.
	Name string

	// FromVersion is the currently installed version.
	FromVersion string

	// ToVersion is the latest available version.
	ToVersion string

	// Skipped is true when the update was not performed.
	Skipped bool

	// SkipReason explains why the update was skipped.
	SkipReason string

	// Manifest is the resolved plugin manifest for the update.
	Manifest *PluginManifest

	// Version is the resolved target version for the update.
	Version *PluginVersion

	// Platform is the resolved platform entry for the update.
	Platform *PluginPlatform

	// Err is set when the update check itself failed.
	Err error
}

// CheckUpdate compares the installed version against the latest in the catalog.
// Returns an UpdateResult describing what should happen. When an update is
// available, Manifest is populated so the caller can install without re-fetching.
func CheckUpdate(pluginDir string, name string, catalog *CatalogClient, goos, goarch string) UpdateResult {
	receipt := ReadReceipt(pluginDir, name)
	if receipt == nil {
		return UpdateResult{
			Name:       name,
			Skipped:    true,
			SkipReason: SkipReasonManual,
		}
	}

	manifest, err := catalog.FetchManifest(name)
	if err != nil {
		return UpdateResult{Name: name, Err: err}
	}

	latest, err := ResolveVersion(manifest, "")
	if err != nil {
		return UpdateResult{Name: name, Err: err}
	}

	if receipt.Version == latest.Version {
		return UpdateResult{
			Name:        name,
			FromVersion: receipt.Version,
			ToVersion:   latest.Version,
			Skipped:     true,
			SkipReason:  SkipReasonUpToDate,
		}
	}

	plat, err := ResolvePlatform(latest, goos, goarch)
	if err != nil {
		return UpdateResult{Name: name, Err: err}
	}

	return UpdateResult{
		Name:        name,
		FromVersion: receipt.Version,
		ToVersion:   latest.Version,
		Manifest:    manifest,
		Version:     latest,
		Platform:    plat,
	}
}
