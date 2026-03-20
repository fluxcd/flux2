/*
Copyright 2022 The Flux authors

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_saveReaderToFile(t *testing.T) {
	g := NewWithT(t)

	testString := `apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp
data:
  foo: bar`

	tests := []struct {
		name      string
		string    string
		expectErr bool
	}{
		{
			name:   "yaml",
			string: testString,
		},
		{
			name:   "yaml with carriage return",
			string: testString + "\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := saveReaderToFile(strings.NewReader(tt.string))
			g.Expect(err).To(BeNil())

			t.Cleanup(func() { _ = os.Remove(tmpFile) })

			b, err := os.ReadFile(tmpFile)
			if tt.expectErr {
				g.Expect(err).To(Not(BeNil()))
				return
			}

			g.Expect(err).To(BeNil())
			g.Expect(string(b)).To(BeEquivalentTo(testString))
		})

	}
}

func Test_resolveSymlinks(t *testing.T) {
	g := NewWithT(t)

	// Create source directory with a real file
	srcDir := t.TempDir()
	realFile := filepath.Join(srcDir, "real.yaml")
	g.Expect(os.WriteFile(realFile, []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: test\n"), 0o644)).To(Succeed())

	// Create a directory with symlinks pointing to files outside it
	symlinkDir := t.TempDir()
	symlinkFile := filepath.Join(symlinkDir, "linked.yaml")
	g.Expect(os.Symlink(realFile, symlinkFile)).To(Succeed())

	// Also add a regular file in the symlink dir
	regularFile := filepath.Join(symlinkDir, "regular.yaml")
	g.Expect(os.WriteFile(regularFile, []byte("apiVersion: v1\nkind: ConfigMap\n"), 0o644)).To(Succeed())

	// Create a symlinked subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	g.Expect(os.MkdirAll(subDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subDir, "nested.yaml"), []byte("nested"), 0o644)).To(Succeed())
	g.Expect(os.Symlink(subDir, filepath.Join(symlinkDir, "linkeddir"))).To(Succeed())

	// Resolve symlinks
	resolved, cleanupDir, err := resolveSymlinks(symlinkDir)
	g.Expect(err).To(BeNil())
	t.Cleanup(func() { os.RemoveAll(cleanupDir) })

	// Verify the regular file was copied
	content, err := os.ReadFile(filepath.Join(resolved, "regular.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(string(content)).To(Equal("apiVersion: v1\nkind: ConfigMap\n"))

	// Verify the symlinked file was resolved and copied
	content, err = os.ReadFile(filepath.Join(resolved, "linked.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(string(content)).To(ContainSubstring("kind: Namespace"))

	// Verify that the resolved file is a regular file, not a symlink
	info, err := os.Lstat(filepath.Join(resolved, "linked.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(info.Mode().IsRegular()).To(BeTrue())

	// Verify that the symlinked directory was resolved and its contents were copied
	content, err = os.ReadFile(filepath.Join(resolved, "linkeddir", "nested.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(string(content)).To(Equal("nested"))

	// Verify that the file inside the symlinked directory is a regular file
	info, err = os.Lstat(filepath.Join(resolved, "linkeddir", "nested.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(info.Mode().IsRegular()).To(BeTrue())
}

func Test_resolveSymlinks_singleFile(t *testing.T) {
	g := NewWithT(t)

	// Create a real file
	srcDir := t.TempDir()
	realFile := filepath.Join(srcDir, "manifest.yaml")
	g.Expect(os.WriteFile(realFile, []byte("kind: ConfigMap"), 0o644)).To(Succeed())

	// Create a symlink to the real file
	linkDir := t.TempDir()
	linkFile := filepath.Join(linkDir, "link.yaml")
	g.Expect(os.Symlink(realFile, linkFile)).To(Succeed())

	// Resolve the single symlinked file
	resolved, cleanupDir, err := resolveSymlinks(linkFile)
	g.Expect(err).To(BeNil())
	t.Cleanup(func() { os.RemoveAll(cleanupDir) })

	// The returned path should be a file, not a directory
	info, err := os.Stat(resolved)
	g.Expect(err).To(BeNil())
	g.Expect(info.IsDir()).To(BeFalse())

	// Verify contents
	content, err := os.ReadFile(resolved)
	g.Expect(err).To(BeNil())
	g.Expect(string(content)).To(Equal("kind: ConfigMap"))
}

func Test_resolveSymlinks_cycle(t *testing.T) {
	g := NewWithT(t)

	// Create a directory with a symlink cycle: dir/link -> dir
	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "file.yaml"), []byte("data"), 0o644)).To(Succeed())
	g.Expect(os.Symlink(dir, filepath.Join(dir, "cycle"))).To(Succeed())

	// resolveSymlinks should not infinite-loop
	resolved, cleanupDir, err := resolveSymlinks(dir)
	g.Expect(err).To(BeNil())
	t.Cleanup(func() { os.RemoveAll(cleanupDir) })

	// The file should be copied
	content, err := os.ReadFile(filepath.Join(resolved, "file.yaml"))
	g.Expect(err).To(BeNil())
	g.Expect(string(content)).To(Equal("data"))

	// The cycle directory should exist but not cause infinite nesting
	_, err = os.Stat(filepath.Join(resolved, "cycle"))
	g.Expect(err).To(BeNil())

	// There should NOT be deeply nested cycle/cycle/cycle/... paths
	_, err = os.Stat(filepath.Join(resolved, "cycle", "cycle", "cycle"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}
