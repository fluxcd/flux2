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
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	pluginPrefix   = "flux-"
	defaultDirName = "plugins"
	defaultBaseDir = ".fluxcd"
	envPluginDir   = "FLUXCD_PLUGINS"
)

// reservedNames are command names that cannot be used as plugin names.
var reservedNames = map[string]bool{
	"plugin": true,
	"help":   true,
}

// Plugin represents a discovered plugin binary.
type Plugin struct {
	Name string // e.g., "operator" (derived from "flux-operator")
	Path string // absolute path to binary
}

// Handler discovers and executes plugins. Uses dependency injection
// for testability.
type Handler struct {
	ReadDir func(name string) ([]os.DirEntry, error)
	Stat    func(name string) (os.FileInfo, error)
	GetEnv  func(key string) string
	HomeDir func() (string, error)
}

// NewHandler returns a Handler with production defaults.
func NewHandler() *Handler {
	return &Handler{
		ReadDir: os.ReadDir,
		Stat:    os.Stat,
		GetEnv:  os.Getenv,
		HomeDir: os.UserHomeDir,
	}
}

// Discover scans the plugin directory for executables matching flux-*.
// It skips builtins, reserved names, directories, non-executable files,
// and broken symlinks.
func (h *Handler) Discover(builtinNames []string) []Plugin {
	dir := h.PluginDir()
	if dir == "" {
		return nil
	}

	entries, err := h.ReadDir(dir)
	if err != nil {
		return nil
	}

	builtins := make(map[string]bool, len(builtinNames))
	for _, name := range builtinNames {
		builtins[name] = true
	}

	var plugins []Plugin
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, pluginPrefix) {
			continue
		}
		if entry.IsDir() {
			continue
		}

		pluginName := pluginNameFromBinary(name)
		if pluginName == "" {
			continue
		}
		if reservedNames[pluginName] || builtins[pluginName] {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// Use Stat to follow symlinks and check the target.
		info, err := h.Stat(fullPath)
		if err != nil {
			// Broken symlink, permission denied, etc.
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if !isExecutable(info) {
			continue
		}

		plugins = append(plugins, Plugin{
			Name: pluginName,
			Path: fullPath,
		})
	}

	return plugins
}

// PluginDir returns the plugin directory path. If FLUXCD_PLUGINS is set,
// returns that path. Otherwise returns ~/.fluxcd/plugins/.
// Does not create the directory — callers that write (install, update)
// should call EnsurePluginDir first.
func (h *Handler) PluginDir() string {
	if dir := h.GetEnv(envPluginDir); dir != "" {
		return dir
	}

	home, err := h.HomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, defaultBaseDir, defaultDirName)
}

// EnsurePluginDir creates the plugin directory if it doesn't exist
// and returns the path. Best-effort — ignores mkdir errors for
// read-only filesystems. User-managed directories (via $FLUXCD_PLUGINS)
// are not auto-created.
func (h *Handler) EnsurePluginDir() string {
	if envDir := h.GetEnv(envPluginDir); envDir != "" {
		return envDir
	}

	home, err := h.HomeDir()
	if err != nil {
		return ""
	}

	dir := filepath.Join(home, defaultBaseDir, defaultDirName)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// pluginNameFromBinary extracts the plugin name from a binary filename.
// "flux-operator" → "operator", "flux-my-tool" → "my-tool".
// Returns empty string for invalid names.
func pluginNameFromBinary(filename string) string {
	if !strings.HasPrefix(filename, pluginPrefix) {
		return ""
	}

	name := strings.TrimPrefix(filename, pluginPrefix)

	// On Windows, strip known extensions.
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".exe", ".cmd", ".bat"} {
			if strings.HasSuffix(strings.ToLower(name), ext) {
				name = name[:len(name)-len(ext)]
				break
			}
		}
	}

	if name == "" {
		return ""
	}

	return name
}

// isExecutable checks if a file has the executable bit set.
// On Windows, this always returns true (executability is determined by extension).
func isExecutable(info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}
