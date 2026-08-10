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

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
	plugintypes "github.com/fluxcd/flux2/v2/pkg/plugin"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query[@<version>]]",
	Short: "Search the plugin catalog",
	Long: `The plugin search command lists available plugins from the Flux plugin catalog.

Examples:
  # List all plugins in the catalog
  flux plugin search

  # List the digests for all plugins
  flux plugin search --digests

  # List the digests of a specific version (implies --digests)
  flux plugin search operator@0.45.0`,
	Args: cobra.MaximumNArgs(1),
	RunE: pluginSearchCmdRun,
}

type pluginSearchFlags struct {
	digests bool
}

var pluginSearchArgs pluginSearchFlags

func init() {
	pluginSearchCmd.Flags().BoolVar(&pluginSearchArgs.digests, "digests", false,
		"list the digest of every platform binary")
	pluginCmd.AddCommand(pluginSearchCmd)
}

func pluginSearchCmdRun(cmd *cobra.Command, args []string) error {
	var arg, query, version string
	if len(args) == 1 {
		arg = args[0]
		query, version = parseNameVersion(arg)
		query = strings.ToLower(query)
	}

	if isDigestRef(version) {
		return fmt.Errorf("searching by digest is not supported, use 'flux plugin install %s' to pin a plugin to a digest", arg)
	}
	if version != "" && query == "" {
		return fmt.Errorf("a query is required in front of '@%s', e.g. 'flux plugin search operator@%[1]s'", version)
	}

	// Print digest information for a given version
	digests := pluginSearchArgs.digests || version != ""

	catalogClient := newCatalogClient()
	catalog, err := catalogClient.FetchCatalog()
	if err != nil {
		return err
	}

	var entries []plugintypes.CatalogEntry
	for _, entry := range catalog.Plugins {
		if query != "" &&
			!strings.Contains(strings.ToLower(entry.Name), query) &&
			!strings.Contains(strings.ToLower(entry.Description), query) {
			continue
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		if arg != "" {
			cmd.Printf("No plugins matching %q found in catalog\n", arg)
		} else {
			cmd.Println("No plugins found in catalog")
		}
		return nil
	}

	if digests {
		trees, warnings := pluginDigestTrees(catalogClient, entries, version)
		for _, w := range warnings {
			logger.Warningf("%s", w)
		}
		if len(trees) == 0 {
			if len(entries) == 1 {
				return fmt.Errorf("failed to fetch digests for plugin %s", entries[0].Name)
			}
			return fmt.Errorf("failed to fetch digests for all %d matching plugins", len(entries))
		}
		printPluginDigestTrees(cmd.OutOrStdout(), trees)
		return nil
	}

	header := []string{"NAME", "DESCRIPTION", "INSTALLED"}
	return printers.TablePrinter(header).Print(cmd.OutOrStdout(), pluginCatalogRows(entries))
}

// pluginCatalogRows returns one row per catalog entry, annotated with the
// installed version when a receipt exists.
func pluginCatalogRows(entries []plugintypes.CatalogEntry) [][]string {
	pluginDir := pluginHandler.PluginDir()

	var rows [][]string
	for _, entry := range entries {
		installed := ""
		if receipt := plugin.ReadReceipt(pluginDir, entry.Name); receipt != nil {
			installed = receipt.Version
		}

		rows = append(rows, []string{entry.Name, entry.Description, installed})
	}

	return rows
}

// pluginDigestTree holds the platform digests of a single plugin version.
type pluginDigestTree struct {
	name        string
	version     string
	description string
	platforms   []plugintypes.Platform
}

// pluginDigestTrees fetches the manifest of every entry and returns one tree
// per plugin with the platform digests of the requested version. Any errors
// encountered when fetching plugin information are recorded and plugins are
// skipped in output.
func pluginDigestTrees(catalogClient *plugin.CatalogClient, entries []plugintypes.CatalogEntry, version string) ([]pluginDigestTree, []error) {
	sp := newPluginSpinner("fetching plugin digests")
	sp.Start()
	defer sp.Stop()

	var trees []pluginDigestTree
	var warnings []error
	for _, entry := range entries {
		manifest, err := catalogClient.FetchManifest(entry.Name)
		if err != nil {
			warnings = append(warnings, err)
			continue
		}

		pv, err := plugin.ResolveVersion(manifest, version)
		if err != nil {
			warnings = append(warnings, err)
			continue
		}

		trees = append(trees, pluginDigestTree{
			name:        entry.Name,
			version:     pv.Version,
			description: entry.Description,
			platforms:   pv.Platforms,
		})
	}

	return trees, warnings
}

// printPluginDigestTrees renders one tree per plugin, with the plugin header
// as the root and one branch per os/arch, digests aligned per plugin.
func printPluginDigestTrees(w io.Writer, trees []pluginDigestTree) {
	for _, t := range trees {
		fmt.Fprintf(w, "%s  %s  %s\n", t.name, t.version, t.description)

		width := 0
		for _, plat := range t.platforms {
			if l := len(plat.OS) + len(plat.Arch) + 1; l > width {
				width = l
			}
		}
		for i, plat := range t.platforms {
			branch := "├──"
			if i == len(t.platforms)-1 {
				branch = "└──"
			}
			fmt.Fprintf(w, "%s %-*s  %s\n", branch, width, plat.OS+"/"+plat.Arch, plat.Checksum)
		}
	}
}
