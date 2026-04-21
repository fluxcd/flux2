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

package plugin

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"
)

// mockDirEntry implements os.DirEntry for testing.
type mockDirEntry struct {
	name  string
	isDir bool
	mode  fs.FileMode
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return m.mode }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// mockFileInfo implements os.FileInfo for testing.
type mockFileInfo struct {
	name    string
	mode    fs.FileMode
	isDir   bool
	regular bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() fs.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any           { return nil }

func newTestHandler(entries []os.DirEntry, statResults map[string]*mockFileInfo, envVars map[string]string) *Handler {
	return &Handler{
		ReadDir: func(name string) ([]os.DirEntry, error) {
			if entries == nil {
				return nil, fmt.Errorf("directory not found")
			}
			return entries, nil
		},
		Stat: func(name string) (os.FileInfo, error) {
			if info, ok := statResults[name]; ok {
				return info, nil
			}
			return nil, fmt.Errorf("file not found: %s", name)
		},
		GetEnv: func(key string) string {
			return envVars[key]
		},
		HomeDir: func() (string, error) {
			return "/home/testuser", nil
		},
	}
}

func TestDiscover(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-operator", mode: 0},
		&mockDirEntry{name: "flux-local", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-operator": {name: "flux-operator", mode: 0o755},
		"/test/plugins/flux-local":    {name: "flux-local", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if plugins[0].Name != "operator" {
		t.Errorf("expected name 'operator', got %q", plugins[0].Name)
	}
	if plugins[1].Name != "local" {
		t.Errorf("expected name 'local', got %q", plugins[1].Name)
	}
}

func TestDiscoverSkipsBuiltins(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-version", mode: 0},
		&mockDirEntry{name: "flux-get", mode: 0},
		&mockDirEntry{name: "flux-operator", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-version":  {name: "flux-version", mode: 0o755},
		"/test/plugins/flux-get":      {name: "flux-get", mode: 0o755},
		"/test/plugins/flux-operator": {name: "flux-operator", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover([]string{"version", "get"})
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "operator" {
		t.Errorf("expected name 'operator', got %q", plugins[0].Name)
	}
}

func TestDiscoverSkipsReserved(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-plugin", mode: 0},
		&mockDirEntry{name: "flux-help", mode: 0},
		&mockDirEntry{name: "flux-operator", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-plugin":   {name: "flux-plugin", mode: 0o755},
		"/test/plugins/flux-help":     {name: "flux-help", mode: 0o755},
		"/test/plugins/flux-operator": {name: "flux-operator", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "operator" {
		t.Errorf("expected name 'operator', got %q", plugins[0].Name)
	}
}

func TestDiscoverSkipsNonExecutable(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-noperm", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-noperm": {name: "flux-noperm", mode: 0o644},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverSkipsDirectories(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-somedir", isDir: true, mode: fs.ModeDir},
	}
	stats := map[string]*mockFileInfo{}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverFollowsSymlinks(t *testing.T) {
	entries := []os.DirEntry{
		// Symlink entry — Type() returns symlink, but Stat resolves to regular executable.
		&mockDirEntry{name: "flux-linked", mode: fs.ModeSymlink},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-linked": {name: "flux-linked", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "linked" {
		t.Errorf("expected name 'linked', got %q", plugins[0].Name)
	}
}

func TestDiscoverDirNotExist(t *testing.T) {
	h := newTestHandler(nil, nil, map[string]string{envPluginDir: "/nonexistent"})

	plugins := h.Discover(nil)
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestDiscoverCustomDir(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-custom", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/custom/path/flux-custom": {name: "flux-custom", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/custom/path"})

	plugins := h.Discover(nil)
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Path != "/custom/path/flux-custom" {
		t.Errorf("expected path '/custom/path/flux-custom', got %q", plugins[0].Path)
	}
}

func TestDiscoverSkipsNonFluxPrefix(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "kubectl-foo", mode: 0},
		&mockDirEntry{name: "random-binary", mode: 0},
		&mockDirEntry{name: "flux-operator", mode: 0},
	}
	stats := map[string]*mockFileInfo{
		"/test/plugins/flux-operator": {name: "flux-operator", mode: 0o755},
	}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
}

func TestDiscoverBrokenSymlink(t *testing.T) {
	entries := []os.DirEntry{
		&mockDirEntry{name: "flux-broken", mode: fs.ModeSymlink},
	}
	// No stat entry for flux-broken — simulates a broken symlink.
	stats := map[string]*mockFileInfo{}
	h := newTestHandler(entries, stats, map[string]string{envPluginDir: "/test/plugins"})

	plugins := h.Discover(nil)
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestPluginNameFromBinary(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"flux-operator", "operator"},
		{"flux-my-tool", "my-tool"},
		{"flux-", ""},
		{"notflux-thing", ""},
		{"flux-a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pluginNameFromBinary(tt.input)
			if got != tt.expected {
				t.Errorf("pluginNameFromBinary(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPluginDir(t *testing.T) {
	t.Run("uses env var", func(t *testing.T) {
		h := &Handler{
			GetEnv: func(key string) string {
				if key == envPluginDir {
					return "/custom/plugins"
				}
				return ""
			},
			HomeDir: func() (string, error) {
				return "/home/user", nil
			},
		}
		dir := h.PluginDir()
		if dir != "/custom/plugins" {
			t.Errorf("expected '/custom/plugins', got %q", dir)
		}
	})

	t.Run("uses default", func(t *testing.T) {
		h := &Handler{
			GetEnv: func(key string) string { return "" },
			HomeDir: func() (string, error) {
				return "/home/user", nil
			},
		}
		dir := h.PluginDir()
		if dir != "/home/user/.fluxcd/plugins" {
			t.Errorf("expected '/home/user/.fluxcd/plugins', got %q", dir)
		}
	})
}
