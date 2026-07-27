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
	"net/http"
	"net/http/httptest"
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

const (
	testCatalogYAML = `apiVersion: cli.fluxcd.io/v1beta1
kind: PluginCatalog
plugins:
  - name: some-tool
    description: Some tool
  - name: my-plugin
    description: My plugin`

	testSomeToolDigestDarwinLatest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	testSomeToolDigestLinuxLatest  = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	testSomeToolDigestDarwinOld    = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	testSomeToolDigestLinuxOld     = "sha256:4444444444444444444444444444444444444444444444444444444444444444"

	testSomeToolYAML = `apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: some-tool
description: Some tool
bin: flux-some-tool
versions:
  - version: 0.12.0
    platforms:
      - os: darwin
        arch: arm64
        url: https://example.com/flux-some-tool_0.12.0_darwin_arm64.tar.gz
        checksum: ` + testSomeToolDigestDarwinLatest + `
      - os: linux
        arch: amd64
        url: https://example.com/flux-some-tool_0.12.0_linux_amd64.tar.gz
        checksum: ` + testSomeToolDigestLinuxLatest + `
  - version: 0.11.0
    platforms:
      - os: darwin
        arch: arm64
        url: https://example.com/flux-some-tool_0.11.0_darwin_arm64.tar.gz
        checksum: ` + testSomeToolDigestDarwinOld + `
      - os: linux
        arch: amd64
        url: https://example.com/flux-some-tool_0.11.0_linux_amd64.tar.gz
        checksum: ` + testSomeToolDigestLinuxOld

	testMyPluginDigest = "sha256:5555555555555555555555555555555555555555555555555555555555555555"

	testMyPluginYAML = `apiVersion: cli.fluxcd.io/v1beta1
kind: Plugin
name: my-plugin
description: My plugin
bin: flux-my-plugin
versions:
  - version: 0.1.2
    platforms:
      - os: linux
        arch: amd64
        url: https://example.com/flux-my-plugin_0.1.2_linux_amd64.tar.gz
        checksum: ` + testMyPluginDigest
)

// serveTestCatalog starts a server for the default plugin catalog fixtures and
// points pluginHandler at it through FLUXCD_PLUGIN_CATALOG.
func serveTestCatalog(t *testing.T) {
	t.Helper()

	files := map[string]string{
		"catalog.yaml":   testCatalogYAML,
		"some-tool.yaml": testSomeToolYAML,
		"my-plugin.yaml": testMyPluginYAML,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	origHandler := pluginHandler
	t.Cleanup(func() { pluginHandler = origHandler })

	pluginDir := t.TempDir()
	pluginHandler = &plugin.Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv: func(key string) string {
			switch key {
			case "FLUXCD_PLUGIN_CATALOG":
				return server.URL + "/"
			case "FLUXCD_PLUGINS":
				return pluginDir
			}
			return ""
		},
		HomeDir: func() (string, error) { return t.TempDir(), nil },
	}
}

func TestPluginSearch(t *testing.T) {
	serveTestCatalog(t)

	tests := []struct {
		name    string
		args    string
		want    []string
		notWant []string
	}{
		{
			name:    "lists the whole catalog",
			args:    "plugin search",
			want:    []string{"NAME", "DESCRIPTION", "INSTALLED", "some-tool", "Some tool", "my-plugin", "My plugin"},
			notWant: []string{"sha256:"},
		},
		{
			name:    "narrows down to the query",
			args:    "plugin search some-tool",
			want:    []string{"some-tool", "Some tool"},
			notWant: []string{"my-plugin", "sha256:"},
		},
		{
			name: "lists the digests of every plugin",
			args: "plugin search --digests",
			want: []string{"NAME", "VERSION", "OS/ARCH", "DIGEST",
				testSomeToolDigestDarwinLatest, testSomeToolDigestLinuxLatest, testMyPluginDigest},
		},
		{
			name: "lists every platform of the latest version",
			args: "plugin search some-tool --digests",
			want: []string{"0.12.0",
				"darwin/arm64", testSomeToolDigestDarwinLatest,
				"linux/amd64", testSomeToolDigestLinuxLatest},
			notWant: []string{"0.11.0", testMyPluginDigest},
		},
		{
			name:    "lists the platforms of a specific version",
			args:    "plugin search some-tool@0.11.0 --digests",
			want:    []string{"0.11.0", testSomeToolDigestDarwinOld, testSomeToolDigestLinuxOld},
			notWant: []string{"0.12.0", testSomeToolDigestDarwinLatest},
		},
		{
			name:    "a version implies --digests",
			args:    "plugin search some-tool@0.11.0",
			want:    []string{"0.11.0", testSomeToolDigestDarwinOld, testSomeToolDigestLinuxOld},
			notWant: []string{"0.12.0", testSomeToolDigestDarwinLatest},
		},
		{
			name:    "reports an unknown version as no match",
			args:    "plugin search some-tool@9.9.9 --digests",
			want:    []string{`No plugins matching "some-tool@9.9.9"`},
			notWant: []string{"sha256:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeCommand(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("expected %q in output, got: %s", want, output)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("expected %q to be absent from output, got: %s", notWant, output)
				}
			}
		})
	}
}

func TestPluginSearchErrors(t *testing.T) {
	serveTestCatalog(t)

	tests := []struct {
		name    string
		args    string
		wantErr string
	}{
		{
			name:    "version without a query",
			args:    "plugin search @0.11.0",
			wantErr: "a query is required",
		},
		{
			name:    "digest instead of a version",
			args:    "plugin search some-tool@sha256:abc123 --digests",
			wantErr: "not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args)
			if err == nil {
				t.Fatalf("expected an error containing %q, got none", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected an error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
