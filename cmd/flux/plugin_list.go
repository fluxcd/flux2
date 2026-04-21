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
	"github.com/spf13/cobra"

	"github.com/fluxcd/flux2/v2/internal/plugin"
	"github.com/fluxcd/flux2/v2/pkg/printers"
)

var pluginListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed plugins",
	Long:    `The plugin list command shows all installed plugins with their versions and paths.`,
	RunE:    pluginListCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
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
