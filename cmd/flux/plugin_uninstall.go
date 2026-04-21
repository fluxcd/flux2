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
)

var pluginUninstallCmd = &cobra.Command{
	Use:     "uninstall <name>",
	Aliases: []string{"delete"},
	Short:   "Uninstall a plugin",
	Long:    `The plugin uninstall command removes a plugin binary and its receipt from the plugin directory.`,
	Args:    cobra.ExactArgs(1),
	RunE:    pluginUninstallCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginUninstallCmd)
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
