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

package kustomization

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/pkg/manifestgen"
)

func Generate(options Options) (*manifestgen.Manifest, error) {
	kfile := filepath.Join(options.TargetPath, konfig.DefaultKustomizationFileName())
	abskfile := filepath.Join(options.BaseDir, kfile)

	scan := func(base string) ([]string, error) {
		var paths []string
		uf := kunstruct.NewKunstructuredFactoryImpl()
		err := options.FileSystem.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == base {
				return nil
			}
			if info.IsDir() {
				// If a sub-directory contains an existing Kustomization file add the
				// directory as a resource and do not decent into it.
				for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
					if options.FileSystem.Exists(filepath.Join(path, kfilename)) {
						paths = append(paths, path)
						return filepath.SkipDir
					}
				}
				return nil
			}
			fContents, err := options.FileSystem.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := uf.SliceFromBytes(fContents); err != nil {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		return paths, err
	}

	if _, err := os.Stat(abskfile); err != nil {
		abs, err := filepath.Abs(filepath.Dir(abskfile))
		if err != nil {
			return nil, err
		}

		files, err := scan(abs)
		if err != nil {
			return nil, err
		}

		f, err := options.FileSystem.Create(abskfile)
		if err != nil {
			return nil, err
		}
		f.Close()

		kus := kustypes.Kustomization{
			TypeMeta: kustypes.TypeMeta{
				APIVersion: kustypes.KustomizationVersion,
				Kind:       kustypes.KustomizationKind,
			},
		}

		var resources []string
		for _, file := range files {
			relP, err := filepath.Rel(abs, file)
			if err != nil {
				return nil, err
			}
			resources = append(resources, relP)
		}

		kus.Resources = resources
		kd, err := yaml.Marshal(kus)
		if err != nil {
			return nil, err
		}

		return &manifestgen.Manifest{
			Path:    kfile,
			Content: string(kd),
		}, nil
	}

	kd, err := ioutil.ReadFile(abskfile)
	if err != nil {
		return nil, err
	}
	return &manifestgen.Manifest{
		Path:    kfile,
		Content: string(kd),
	}, nil
}
