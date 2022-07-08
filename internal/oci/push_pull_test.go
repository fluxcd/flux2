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

package oci

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	. "github.com/onsi/gomega"
)

func Test_Push_Pull(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	tag := "v0.0.1"
	repo := "test-push" + randStringRunes(5)

	url := fmt.Sprintf("%s/%s:%s", dockerReg, repo, tag)
	metadata := Metadata{
		Source:   "github.com/fluxcd/fluxv2",
		Revision: "rev",
	}

	testDir := "testdata/build"
	_, err := Push(ctx, url, testDir, metadata)
	g.Expect(err).ToNot(HaveOccurred())

	tags, err := crane.ListTags(fmt.Sprintf("%s/%s", dockerReg, repo))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(tags)).To(BeEquivalentTo(1))

	tmpDir := t.TempDir()
	_, err = Pull(ctx, url, tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Walk directory the test directory and check that all paths exists in the extracted archive
	err = filepath.Walk(testDir, func(path string, info fs.FileInfo, err error) error {
		tmpPath := filepath.Join(tmpDir, path)
		if _, err := os.Stat(tmpPath); err != nil && os.IsNotExist(err) {
			return fmt.Errorf("path '%s' doesn't exist in archive", path)
		}

		return nil
	})
}
