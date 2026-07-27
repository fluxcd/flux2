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

	var header []string
	var rows [][]string
	if digests {
		header = []string{"NAME", "VERSION", "OS/ARCH", "DIGEST"}
		rows, err = pluginDigestRows(catalogClient, entries, version)
	} else {
		header = []string{"NAME", "DESCRIPTION", "INSTALLED"}
		rows = pluginCatalogRows(entries)
	}
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		if arg != "" {
			cmd.Printf("No plugins matching %q found in catalog\n", arg)
		} else {
			cmd.Println("No plugins found in catalog")
		}
		return nil
	}

	return printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
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

// pluginDigestRows fetches the manifest of every entry and returns one row per
// os/arch for the binary of the requested version.
func pluginDigestRows(catalogClient *plugin.CatalogClient, entries []plugintypes.CatalogEntry, version string) ([][]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	sp := newPluginSpinner("fetching plugin digests")
	sp.Start()
	defer sp.Stop()

	var rows [][]string
	for _, entry := range entries {
		manifest, err := catalogClient.FetchManifest(entry.Name)
		if err != nil {
			return nil, err
		}

		pv, err := plugin.ResolveVersion(manifest, version)
		if err != nil {
			continue
		}

		for _, plat := range pv.Platforms {
			rows = append(rows, []string{
				entry.Name,
				pv.Version,
				fmt.Sprintf("%s/%s", plat.OS, plat.Arch),
				plat.Checksum,
			})
		}
	}

	return rows, nil
}
