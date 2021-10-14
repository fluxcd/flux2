/*
Copyright 2021 The Flux authors

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
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
)

//go:embed manifests/*.yaml
var embeddedManifests embed.FS

func writeEmbeddedManifests(dir string) error {
	manifests, err := fs.ReadDir(embeddedManifests, "manifests")
	if err != nil {
		return err
	}
	for _, manifest := range manifests {
		data, err := fs.ReadFile(embeddedManifests, path.Join("manifests", manifest.Name()))
		if err != nil {
			return fmt.Errorf("reading file failed: %w", err)
		}

		err = os.WriteFile(path.Join(dir, manifest.Name()), data, 0666)
		if err != nil {
			return fmt.Errorf("writing file failed: %w", err)
		}
	}
	return nil
}
