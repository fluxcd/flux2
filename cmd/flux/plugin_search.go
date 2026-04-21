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
	"strings"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the plugin catalog",
	Long:  `The plugin search command lists available plugins from the Flux plugin catalog.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  pluginSearchCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginSearchCmd)
}

func pluginSearchCmdRun(cmd *cobra.Command, args []string) error {
	catalogClient := newCatalogClient()
	catalog, err := catalogClient.FetchCatalog()
	if err != nil {
		return err
	}

	var query string
	if len(args) == 1 {
		query = strings.ToLower(args[0])
	}

	pluginDir := pluginHandler.PluginDir()
	header := []string{"NAME", "DESCRIPTION", "INSTALLED"}
	var rows [][]string
	for _, entry := range catalog.Plugins {
		if query != "" {
			if !strings.Contains(strings.ToLower(entry.Name), query) &&
				!strings.Contains(strings.ToLower(entry.Description), query) {
				continue
			}
		}

		installed := ""
		if receipt := plugin.ReadReceipt(pluginDir, entry.Name); receipt != nil {
			installed = receipt.Version
		}

		rows = append(rows, []string{entry.Name, entry.Description, installed})
	}

	if len(rows) == 0 {
		if query != "" {
			cmd.Printf("No plugins matching %q found in catalog\n", query)
		} else {
			cmd.Println("No plugins found in catalog")
		}
		return nil
	}

	return printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
}
