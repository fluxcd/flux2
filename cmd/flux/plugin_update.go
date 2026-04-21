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
	"runtime"

	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
)

var pluginUpdateCmd = &cobra.Command{
	Use:     "update [name]",
	Aliases: []string{"upgrade"},
	Short:   "Update installed plugins",
	Long: `The plugin update command updates installed plugins to their latest versions.

Examples:
  # Update a single plugin
  flux plugin update operator

  # Update all installed plugins
  flux plugin update`,
	Args: cobra.MaximumNArgs(1),
	RunE: pluginUpdateCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginUpdateCmd)
}

func pluginUpdateCmdRun(cmd *cobra.Command, args []string) error {
	catalogClient := newCatalogClient()

	plugins := pluginHandler.Discover(builtinCommandNames())
	if len(plugins) == 0 {
		cmd.Println("No plugins found")
		return nil
	}

	// If a specific plugin is requested, filter to just that one.
	if len(args) == 1 {
		name := args[0]
		var found bool
		for _, p := range plugins {
			if p.Name == name {
				plugins = []plugin.Plugin{p}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("plugin %q is not installed", name)
		}
	}

	pluginDir := pluginHandler.EnsurePluginDir()
	installer := plugin.NewInstaller()
	for _, p := range plugins {
		result := plugin.CheckUpdate(pluginDir, p.Name, catalogClient, runtime.GOOS, runtime.GOARCH)
		if result.Err != nil {
			logger.Failuref("error checking %s: %v", p.Name, result.Err)
			continue
		}
		if result.Skipped {
			if result.SkipReason == plugin.SkipReasonManual {
				logger.Warningf("skipping %s (%s)", p.Name, result.SkipReason)
			} else {
				logger.Successf("%s already up to date (v%s)", p.Name, result.FromVersion)
			}
			continue
		}

		sp := newPluginSpinner(fmt.Sprintf("updating %s v%s → v%s", p.Name, result.FromVersion, result.ToVersion))
		sp.Start()
		if err := installer.Install(pluginDir, result.Manifest, result.Version, result.Platform); err != nil {
			sp.Stop()
			logger.Failuref("error updating %s: %v", p.Name, err)
			continue
		}
		sp.Stop()
		logger.Successf("updated %s v%s → v%s", p.Name, result.FromVersion, result.ToVersion)
	}

	return nil
}
