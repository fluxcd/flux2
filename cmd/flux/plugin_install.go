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
	plugintypes "github.com/fluxcd/flux2/v2/pkg/plugin"
)

var pluginInstallCmd = &cobra.Command{
	Use:   "install <name>[@<version>|@<digest>]",
	Short: "Install a plugin from the catalog",
	Long: `The plugin install command downloads and installs a plugin from the Flux plugin catalog.

Examples:
  # Install the latest version
  flux plugin install operator

  # Install a specific version
  flux plugin install operator@0.45.0

  # Install pinned to a specific digest
  flux plugin install operator@sha256:06e0a38db4fa6bc9f705a577c7e58dc020bfe2618e45488599e5ef7bb62e3a8a`,
	Args: cobra.ExactArgs(1),
	RunE: pluginInstallCmdRun,
}

func init() {
	pluginCmd.AddCommand(pluginInstallCmd)
}

func pluginInstallCmdRun(cmd *cobra.Command, args []string) error {
	nameVersion := args[0]
	name, ref := parseNameVersion(nameVersion)

	catalogClient := newCatalogClient()
	manifest, err := catalogClient.FetchManifest(name)
	if err != nil {
		return err
	}

	var pv *plugintypes.Version
	var plat *plugintypes.Platform

	if isDigestRef(ref) {
		dm, err := plugin.ResolveByDigest(manifest, ref, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return err
		}
		pv = dm.Version
		plat = dm.Platform
	} else {
		pv, err = plugin.ResolveVersion(manifest, ref)
		if err != nil {
			return err
		}

		plat, err = plugin.ResolvePlatform(pv, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return fmt.Errorf("plugin %q v%s has no binary for %s/%s", name, pv.Version, runtime.GOOS, runtime.GOARCH)
		}
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
