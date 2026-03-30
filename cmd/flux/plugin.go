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
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
	"github.com/fluxcd/flux2/v2/pkg/printers"
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

var pluginListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed plugins",
	Long:    `The plugin list command shows all installed plugins with their versions and paths.`,
	RunE:    pluginListCmdRun,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <name>[@<version>]",
	Short: "Install a plugin from the catalog",
	Long: `The plugin install command downloads and installs a plugin from the Flux plugin catalog.

Examples:
  # Install the latest version
  flux plugin install operator

  # Install a specific version
  flux plugin install operator@0.45.0`,
	Args: cobra.ExactArgs(1),
	RunE: pluginInstallCmdRun,
}

var pluginUninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: "Uninstall a plugin",
	Long:  `The plugin uninstall command removes a plugin binary and its receipt from the plugin directory.`,
	Args:  cobra.ExactArgs(1),
	RunE:  pluginUninstallCmdRun,
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update installed plugins",
	Long: `The plugin update command updates installed plugins to their latest versions.

Examples:
  # Update a single plugin
  flux plugin update operator

  # Update all installed plugins
  flux plugin update`,
	Args: cobra.MaximumNArgs(1),
	RunE: pluginUpdateCmdRun,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search the plugin catalog",
	Long:  `The plugin search command lists available plugins from the Flux plugin catalog.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  pluginSearchCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
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

func pluginListCmdRun(cmd *cobra.Command, args []string) error {
	pluginDir := pluginHandler.PluginDir()
	plugins := pluginHandler.Discover(builtinCommandNames())
	if len(plugins) == 0 {
		cmd.Println("No plugins found")
		return nil
	}

	header := []string{"NAME", "VERSION", "PATH"}
	var rows [][]string
	for _, p := range plugins {
		version := "manual"
		if receipt := plugin.ReadReceipt(pluginDir, p.Name); receipt != nil {
			version = receipt.Version
		}
		rows = append(rows, []string{p.Name, version, p.Path})
	}

	return printers.TablePrinter(header).Print(cmd.OutOrStdout(), rows)
}

func pluginInstallCmdRun(cmd *cobra.Command, args []string) error {
	nameVersion := args[0]
	name, version := parseNameVersion(nameVersion)

	catalogClient := newCatalogClient()
	manifest, err := catalogClient.FetchManifest(name)
	if err != nil {
		return err
	}

	pv, err := plugin.ResolveVersion(manifest, version)
	if err != nil {
		return err
	}

	plat, err := plugin.ResolvePlatform(pv, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("plugin %q v%s has no binary for %s/%s", name, pv.Version, runtime.GOOS, runtime.GOARCH)
	}

	pluginDir := pluginHandler.EnsurePluginDir()

	installer := plugin.NewInstaller()
	sp := newPluginSpinner(fmt.Sprintf("installing %s v%s", name, pv.Version))
	sp.Start()
	if err := installer.Install(pluginDir, manifest, pv, plat); err != nil {
		sp.Stop()
		return err
	}
	sp.Stop()

	logger.Successf("installed %s v%s", name, pv.Version)
	return nil
}

func pluginUninstallCmdRun(cmd *cobra.Command, args []string) error {
	name := args[0]
	pluginDir := pluginHandler.PluginDir()

	if err := plugin.Uninstall(pluginDir, name); err != nil {
		return err
	}

	logger.Successf("uninstalled %s", name)
	return nil
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
