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
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
)

var pluginHandler = plugin.NewHandler()

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Flux CLI plugins",
	Long:  `The plugin sub-commands manage Flux CLI plugins.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// No-op: skip root's namespace DNS validation for plugin commands.
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pluginCmd)
}

// builtinCommandNames returns the names of all non-plugin commands on rootCmd.
func builtinCommandNames() []string {
	var names []string
	for _, c := range rootCmd.Commands() {
		if c.GroupID != "plugin" {
			names = append(names, c.Name())
		}
	}
	return names
}

// registerPlugins scans the plugin directory and registers discovered
// plugins as Cobra subcommands on rootCmd.
func registerPlugins() {
	plugins := pluginHandler.Discover(builtinCommandNames())
	if len(plugins) == 0 {
		return
	}

	if !rootCmd.ContainsGroup("plugin") {
		rootCmd.AddGroup(&cobra.Group{
			ID:    "plugin",
			Title: "Plugin Commands:",
		})
	}

	for _, p := range plugins {
		cmd := &cobra.Command{
			Use:                p.Name,
			Short:              fmt.Sprintf("Runs the %s plugin", p.Name),
			Long:               fmt.Sprintf("This command runs the %s plugin.\nUse 'flux %s --help' for full plugin help.", p.Name, p.Name),
			DisableFlagParsing: true,
			GroupID:            "plugin",
			ValidArgsFunction:  plugin.CompleteFunc(p.Path),
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				return plugin.Exec(p.Path, args)
			},
		}
		rootCmd.AddCommand(cmd)
	}
}

// parseNameVersion splits "operator@0.45.0" into ("operator", "0.45.0").
// If no @ is present, version is empty (latest).
func parseNameVersion(s string) (string, string) {
	name, version, found := strings.Cut(s, "@")
	if found {
		return name, version
	}
	return s, ""
}

// newCatalogClient creates a CatalogClient that respects FLUXCD_PLUGIN_CATALOG.
func newCatalogClient() *plugin.CatalogClient {
	client := plugin.NewCatalogClient()
	client.GetEnv = pluginHandler.GetEnv
	return client
}

func newPluginSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	return s
}
