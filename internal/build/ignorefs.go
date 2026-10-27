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

package build

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/fluxcd/pkg/sourceignore"
	"github.com/fluxcd/pkg/sourceignore/gitignore"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// ignoreFilterFS decorates a filesys.FileSystem so that every kustomization
// file it serves, not just the one at the build root, has its resources,
// components and crds entries filtered against the --ignore-paths patterns.
//
// fluxcd/pkg/kustomize only filters the kustomization.yaml it generates at
// the build root. Native Kustomize then resolves any referenced base or
// component directly off the filesystem, so a file ignored at the root
// leaks back in whenever it is reachable through a base or sub-directory
// (see fluxcd/flux2#6067). Wrapping the filesystem lets every
// kustomization.yaml Kustomize loads, at any depth and on either side of
// the build root, go through the same filtering as the root file.
type ignoreFilterFS struct {
	filesys.FileSystem
	ignore string
}

// newIgnoreFilterFS wraps fs so that ignore-paths patterns are applied to
// every kustomization file it serves. It returns fs unchanged when there
// are no patterns to apply.
func newIgnoreFilterFS(fs filesys.FileSystem, ignore []string) filesys.FileSystem {
	if len(ignore) == 0 {
		return fs
	}
	return &ignoreFilterFS{FileSystem: fs, ignore: strings.Join(ignore, "\n")}
}

// ReadFile filters ignored entries out of kustomization files as they are
// read. Non-kustomization files, and kustomization files that fail to
// parse, are returned unmodified so Kustomize can report its own error.
func (fs *ignoreFilterFS) ReadFile(path string) ([]byte, error) {
	data, err := fs.FileSystem.ReadFile(path)
	if err != nil || !isKustomizationFile(path) {
		return data, err
	}

	filtered, err := filterIgnoredEntries(fs.FileSystem, path, data, fs.ignore)
	if err != nil {
		return data, nil
	}
	return filtered, nil
}

func isKustomizationFile(path string) bool {
	base := filepath.Base(path)
	for _, name := range konfig.RecognizedKustomizationFileNames() {
		if base == name {
			return true
		}
	}
	return false
}

// filterIgnoredEntries removes resources, components and crds entries that
// match the ignore patterns from the kustomization file at path. Patterns
// are anchored to the file's own directory, not the build root, so a
// pattern such as "**/*.enc.yaml" keeps matching regardless of how deep the
// file sits in the build tree, including outside the build root (e.g. a
// base referenced with "../../base").
func filterIgnoredEntries(fs filesys.FileSystem, path string, data []byte, ignore string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", path, err)
	}
	dir := filepath.Dir(absPath)
	domain := strings.Split(dir, string(filepath.Separator))
	patterns := sourceignore.ReadPatterns(strings.NewReader(ignore), domain)
	matcher := sourceignore.NewMatcher(patterns)

	var kus kustypes.Kustomization
	if err := yaml.Unmarshal(data, &kus); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", path, err)
	}

	var changed bool
	if filterEntries(fs, dir, &kus.Resources, matcher) {
		changed = true
	}
	if filterEntries(fs, dir, &kus.Components, matcher) {
		changed = true
	}
	if filterEntries(fs, dir, &kus.Crds, matcher) {
		changed = true
	}
	if !changed {
		return data, nil
	}

	out, err := yaml.Marshal(&kus)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered %s: %w", path, err)
	}
	return out, nil
}

// filterEntries drops entries from *entries that match the ignore matcher,
// mirroring fluxcd/pkg/kustomize's own root-level filterSlice. It reports
// whether any entry was removed.
func filterEntries(fs filesys.FileSystem, dir string, entries *[]string, matcher gitignore.Matcher) bool {
	start := 0
	changed := false
	for _, entry := range *entries {
		if isRemoteEntry(entry) {
			(*entries)[start] = entry
			start++
			continue
		}

		p := filepath.Join(dir, entry)
		if !fs.Exists(p) {
			// Leave unresolved entries alone; Kustomize will report its own
			// "does not exist" error for them.
			(*entries)[start] = entry
			start++
			continue
		}

		if matcher.Match(strings.Split(p, string(filepath.Separator)), fs.IsDir(p)) {
			changed = true
			continue
		}
		(*entries)[start] = entry
		start++
	}
	*entries = (*entries)[:start]
	return changed
}

// isRemoteEntry mirrors fluxcd/pkg/kustomize's isUrl check: URL resources
// are never local files, so ignore-paths filtering does not apply to them.
func isRemoteEntry(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != ""
}
