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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Build archives the given directory as a tarball to the given local path.
// While archiving, any environment specific data (for example, the user and group name) is stripped from file headers.
func Build(artifactPath, sourceDir string) (err error) {
	if f, err := os.Stat(sourceDir); os.IsNotExist(err) || !f.IsDir() {
		return fmt.Errorf("invalid source dir path: %s", sourceDir)
	}

	tf, err := os.CreateTemp(filepath.Split(sourceDir))
	if err != nil {
		return err
	}
	tmpName := tf.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	sz := &writeCounter{}
	mw := io.MultiWriter(tf, sz)

	gw := gzip.NewWriter(mw)
	tw := tar.NewWriter(gw)
	if err := filepath.Walk(sourceDir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore anything that is not a file or directories e.g. symlinks
		if m := fi.Mode(); !(m.IsRegular() || m.IsDir()) {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, p)
		if err != nil {
			return err
		}
		// The name needs to be modified to maintain directory structure
		// as tar.FileInfoHeader only has access to the base name of the file.
		// Ref: https://golang.org/src/archive/tar/common.go?#L626
		relFilePath := p
		if filepath.IsAbs(sourceDir) {
			relFilePath, err = filepath.Rel(sourceDir, p)
			if err != nil {
				return err
			}
		}
		header.Name = relFilePath

		// Remove any environment specific data.
		header.Gid = 0
		header.Uid = 0
		header.Uname = ""
		header.Gname = ""
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			f.Close()
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	}); err != nil {
		tw.Close()
		gw.Close()
		tf.Close()
		return err
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		tf.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		tf.Close()
		return err
	}
	if err := tf.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpName, 0o640); err != nil {
		return err
	}

	return os.Rename(tmpName, artifactPath)
}

type writeCounter struct {
	written int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.written += int64(n)
	return n, nil
}
