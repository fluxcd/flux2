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
	"os"
	"strings"
	"testing"

	"github.com/fluxcd/flux2/v2/internal/plugin"
)

func TestPluginAppearsInHelp(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginDir := t.TempDir()

	fakeBin := pluginDir + "/flux-testplugin"
	os.WriteFile(fakeBin, []byte("#!/bin/sh\necho test"), 0o755)

	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	registerPlugins()
	defer func() {
		cmds := rootCmd.Commands()
		for _, cmd := range cmds {
			if cmd.Name() == "testplugin" {
				rootCmd.RemoveCommand(cmd)
				break
			}
		}
	}()

	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Plugin Commands:") {
		t.Error("expected 'Plugin Commands:' in help output")
	}
	if !strings.Contains(output, "testplugin") {
		t.Error("expected 'testplugin' in help output")
	}
}

func TestPluginListOutput(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginDir := t.TempDir()

	fakeBin := pluginDir + "/flux-myplugin"
	os.WriteFile(fakeBin, []byte("#!/bin/sh\necho test"), 0o755)

	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	output, err := executeCommand("plugin list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "myplugin") {
		t.Errorf("expected 'myplugin' in output, got: %s", output)
	}
	if !strings.Contains(output, "manual") {
		t.Errorf("expected 'manual' in output (no receipt), got: %s", output)
	}
}

func TestPluginListWithReceipt(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginDir := t.TempDir()

	fakeBin := pluginDir + "/flux-myplugin"
	os.WriteFile(fakeBin, []byte("#!/bin/sh\necho test"), 0o755)
	receipt := pluginDir + "/flux-myplugin.yaml"
	os.WriteFile(receipt, []byte("name: myplugin\nversion: \"1.2.3\"\n"), 0o644)

	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	output, err := executeCommand("plugin list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "1.2.3") {
		t.Errorf("expected version '1.2.3' in output, got: %s", output)
	}
}

func TestPluginListEmpty(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginDir := t.TempDir()

	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	output, err := executeCommand("plugin list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "No plugins found") {
		t.Errorf("expected 'No plugins found', got: %s", output)
	}
}

func TestNoPluginsNoRegistration(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginHandler = &plugin.Handler{
		ReadDir: func(name string) ([]os.DirEntry, error) {
			return nil, fmt.Errorf("no dir")
		},
		Stat: os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return "/nonexistent"
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	// Verify that registerPlugins with no plugins doesn't add any commands.
	before := len(rootCmd.Commands())
	registerPlugins()
	after := len(rootCmd.Commands())
	if after != before {
		t.Errorf("expected no new commands, got %d new", after-before)
	}
}

func TestPluginSkipsPersistentPreRun(t *testing.T) {
	// Plugin commands override root's PersistentPreRunE with a no-op,
	// so an invalid namespace should not trigger a validation error.
	_, err := executeCommand("plugin list")
	if err != nil {
		t.Fatalf("plugin list should not trigger root's namespace validation: %v", err)
	}
}

func TestParseNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"operator", "operator", ""},
		{"operator@0.45.0", "operator", "0.45.0"},
		{"my-tool@1.0.0", "my-tool", "1.0.0"},
		{"plugin@", "plugin", ""},
		{"operator@sha256:abc123", "operator", "sha256:abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := parseNameVersion(tt.input)
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version: got %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestIsDigestRef(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"sha256:06e0a38db4fa6bc9f705a577c7e58dc020bfe2618e45488599e5ef7bb62e3a8a", true},
		{"0.45.0", false},
		{"", false},
		{"sha256", false},
		{"SHA256:abc", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isDigestRef(tt.input); got != tt.want {
				t.Errorf("isDigestRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPluginDiscoverSkipsBuiltins(t *testing.T) {
	origHandler := pluginHandler
	defer func() { pluginHandler = origHandler }()

	pluginDir := t.TempDir()

	for _, name := range []string{"flux-get", "flux-create", "flux-version"} {
		os.WriteFile(pluginDir+"/"+name, []byte("#!/bin/sh"), 0o755)
	}
	os.WriteFile(pluginDir+"/flux-myplugin", []byte("#!/bin/sh"), 0o755)

	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			if key == "FLUXCD_PLUGINS" {
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}

	plugins := pluginHandler.Discover(builtinCommandNames())

	if len(plugins) != 1 {
		names := make([]string, len(plugins))
		for i, p := range plugins {
			names[i] = p.Name
		}
		t.Fatalf("expected 1 plugin, got %d: %v", len(plugins), names)
	}
	if plugins[0].Name != "myplugin" {
		t.Errorf("expected 'myplugin', got %q", plugins[0].Name)
	}
}
