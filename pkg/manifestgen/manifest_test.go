//go:build !windows
// +build !windows

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

package manifestgen

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestManifestWriteFilePermissions(t *testing.T) {
	// Neutralize the ambient umask so the modes requested by WriteFile are
	// observed verbatim. Without this the defect is masked under a typical
	// 022 umask but surfaces under the 0 umask common in CI containers.
	old := syscall.Umask(0)
	defer syscall.Umask(old)

	m := &Manifest{
		Path:    "sub/dir/gotk-sync.yaml",
		Content: "apiVersion: v1\nkind: ConfigMap\n",
	}

	output, err := m.WriteFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	fileInfo, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o644 {
		t.Errorf("manifest file mode = %o, want 0644", got)
	}

	dirInfo, err := os.Stat(filepath.Dir(output))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o755 {
		t.Errorf("manifest dir mode = %o, want 0755", got)
	}
}
