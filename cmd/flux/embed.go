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
