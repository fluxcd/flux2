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

package kustomization

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/pkg/apis/kustomize"
	"github.com/hashicorp/go-multierror"
)

const (
	specField                 = "spec"
	targetNSField             = "targetNamespace"
	patchesField              = "patches"
	patchesSMField            = "patchesStrategicMerge"
	patchesJson6902Field      = "patchesJson6902"
	imagesField               = "images"
	originalKustomizationFile = "kustomization.yaml.original"
)

type action string

const (
	createdAction   action = "created"
	unchangedAction action = "unchanged"
)

type KustomizeGenerator struct {
	kustomization unstructured.Unstructured
}

type SavingOptions func(dirPath, file string, action action) error

func NewGenerator(kustomization unstructured.Unstructured) *KustomizeGenerator {
	return &KustomizeGenerator{
		kustomization: kustomization,
	}
}

func WithSaveOriginalKustomization() SavingOptions {
	return func(dirPath, kfile string, action action) error {
		// copy the original kustomization.yaml to the directory if we did not create it
		if action != createdAction {
			if err := copyFile(kfile, filepath.Join(dirPath, originalKustomizationFile)); err != nil {
				errf := CleanDirectory(dirPath, action)
				return fmt.Errorf("%v %v", err, errf)
			}
		}
		return nil
	}
}

// WriteFile generates a kustomization.yaml in the given directory if it does not exist.
// It apply the flux kustomize resources to the kustomization.yaml and then write the
// updated kustomization.yaml to the directory.
// It returns an action that indicates if the kustomization.yaml was created or not.
// It is the caller responsability to clean up the directory by use the provided function CleanDirectory.
// example:
// err := CleanDirectory(dirPath, action)
// if err != nil {
// 	log.Fatal(err)
// }
func (kg *KustomizeGenerator) WriteFile(dirPath string, opts ...SavingOptions) (action, error) {
	action, err := kg.generateKustomization(dirPath)
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%v %v", err, errf)
	}

	kfile := filepath.Join(dirPath, konfig.DefaultKustomizationFileName())

	data, err := os.ReadFile(kfile)
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%w %s", err, errf)
	}

	kus := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
	}

	if err := yaml.Unmarshal(data, &kus); err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%v %v", err, errf)
	}

	tg, ok, err := kg.getNestedString(specField, targetNSField)
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%v %v", err, errf)
	}
	if ok {
		kus.Namespace = tg
	}

	patches, err := kg.getPatches()
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("unable to get patches: %w", fmt.Errorf("%v %v", err, errf))
	}

	for _, p := range patches {
		kus.Patches = append(kus.Patches, kustypes.Patch{
			Patch:  p.Patch,
			Target: adaptSelector(&p.Target),
		})
	}

	patchesSM, err := kg.getPatchesStrategicMerge()
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("unable to get patchesStrategicMerge: %w", fmt.Errorf("%v %v", err, errf))
	}

	for _, p := range patchesSM {
		kus.PatchesStrategicMerge = append(kus.PatchesStrategicMerge, kustypes.PatchStrategicMerge(p.Raw))
	}

	patchesJSON, err := kg.getPatchesJson6902()
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("unable to get patchesJson6902: %w", fmt.Errorf("%v %v", err, errf))
	}

	for _, p := range patchesJSON {
		patch, err := json.Marshal(p.Patch)
		if err != nil {
			errf := CleanDirectory(dirPath, action)
			return action, fmt.Errorf("%v %v", err, errf)
		}
		kus.PatchesJson6902 = append(kus.PatchesJson6902, kustypes.Patch{
			Patch:  string(patch),
			Target: adaptSelector(&p.Target),
		})
	}

	images, err := kg.getImages()
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("unable to get images: %w", fmt.Errorf("%v %v", err, errf))
	}

	for _, image := range images {
		newImage := kustypes.Image{
			Name:    image.Name,
			NewName: image.NewName,
			NewTag:  image.NewTag,
		}
		if exists, index := checkKustomizeImageExists(kus.Images, image.Name); exists {
			kus.Images[index] = newImage
		} else {
			kus.Images = append(kus.Images, newImage)
		}
	}

	manifest, err := yaml.Marshal(kus)
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%v %v", err, errf)
	}

	// copy the original kustomization.yaml to the directory if we did not create it
	for _, opt := range opts {
		if err := opt(dirPath, kfile, action); err != nil {
			return action, fmt.Errorf("failed to save original kustomization.yaml: %w", err)
		}
	}

	err = os.WriteFile(kfile, manifest, os.ModePerm)
	if err != nil {
		errf := CleanDirectory(dirPath, action)
		return action, fmt.Errorf("%v %v", err, errf)
	}

	return action, nil
}

func (kg *KustomizeGenerator) getPatches() ([]kustomize.Patch, error) {
	patches, ok, err := kg.getNestedSlice(specField, patchesField)
	if err != nil {
		return nil, err
	}

	var resultErr error
	if ok {
		res := make([]kustomize.Patch, 0, len(patches))
		for k, p := range patches {
			patch, ok := p.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("unable to convert patch %d to map[string]interface{}", k)
				resultErr = multierror.Append(resultErr, err)
			}
			var kpatch kustomize.Patch
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(patch, &kpatch)
			if err != nil {
				resultErr = multierror.Append(resultErr, err)
			}
			res = append(res, kpatch)
		}
		return res, resultErr
	}

	return nil, resultErr

}

func (kg *KustomizeGenerator) getPatchesStrategicMerge() ([]apiextensionsv1.JSON, error) {
	patches, ok, err := kg.getNestedSlice(specField, patchesSMField)
	if err != nil {
		return nil, err
	}

	var resultErr error
	if ok {
		res := make([]apiextensionsv1.JSON, 0, len(patches))
		for k, p := range patches {
			patch, ok := p.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("unable to convert patch %d to map[string]interface{}", k)
				resultErr = multierror.Append(resultErr, err)
			}
			var kpatch apiextensionsv1.JSON
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(patch, &kpatch)
			if err != nil {
				resultErr = multierror.Append(resultErr, err)
			}
			res = append(res, kpatch)
		}
		return res, resultErr
	}

	return nil, resultErr

}

func (kg *KustomizeGenerator) getPatchesJson6902() ([]kustomize.JSON6902Patch, error) {
	patches, ok, err := kg.getNestedSlice(specField, patchesJson6902Field)
	if err != nil {
		return nil, err
	}

	var resultErr error
	if ok {
		res := make([]kustomize.JSON6902Patch, 0, len(patches))
		for k, p := range patches {
			patch, ok := p.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("unable to convert patch %d to map[string]interface{}", k)
				resultErr = multierror.Append(resultErr, err)
			}
			var kpatch kustomize.JSON6902Patch
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(patch, &kpatch)
			if err != nil {
				resultErr = multierror.Append(resultErr, err)
			}
			res = append(res, kpatch)
		}
		return res, resultErr
	}

	return nil, resultErr

}

func (kg *KustomizeGenerator) getImages() ([]kustomize.Image, error) {
	img, ok, err := kg.getNestedSlice(specField, imagesField)
	if err != nil {
		return nil, err
	}

	var resultErr error
	if ok {
		res := make([]kustomize.Image, 0, len(img))
		for k, i := range img {
			im, ok := i.(map[string]interface{})
			if !ok {
				err := fmt.Errorf("unable to convert patch %d to map[string]interface{}", k)
				resultErr = multierror.Append(resultErr, err)
			}
			var image kustomize.Image
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(im, &image)
			if err != nil {
				resultErr = multierror.Append(resultErr, err)
			}
			res = append(res, image)
		}
		return res, resultErr
	}

	return nil, resultErr

}

func checkKustomizeImageExists(images []kustypes.Image, imageName string) (bool, int) {
	for i, image := range images {
		if imageName == image.Name {
			return true, i
		}
	}

	return false, -1
}

func (kg *KustomizeGenerator) getNestedString(fields ...string) (string, bool, error) {
	val, ok, err := unstructured.NestedString(kg.kustomization.Object, fields...)
	if err != nil {
		return "", ok, err
	}

	return val, ok, nil
}

func (kg *KustomizeGenerator) getNestedSlice(fields ...string) ([]interface{}, bool, error) {
	val, ok, err := unstructured.NestedSlice(kg.kustomization.Object, fields...)
	if err != nil {
		return nil, ok, err
	}

	return val, ok, nil
}

func (kg *KustomizeGenerator) generateKustomization(dirPath string) (action, error) {
	fs := filesys.MakeFsOnDisk()

	// Determine if there already is a Kustomization file at the root,
	// as this means we do not have to generate one.
	for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
		if kpath := filepath.Join(dirPath, kfilename); fs.Exists(kpath) && !fs.IsDir(kpath) {
			return unchangedAction, nil
		}
	}

	scan := func(base string) ([]string, error) {
		var paths []string
		pvd := provider.NewDefaultDepProvider()
		rf := pvd.GetResourceFactory()
		err := fs.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == base {
				return nil
			}
			if info.IsDir() {
				// If a sub-directory contains an existing kustomization file add the
				// directory as a resource and do not decend into it.
				for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
					if kpath := filepath.Join(path, kfilename); fs.Exists(kpath) && !fs.IsDir(kpath) {
						paths = append(paths, path)
						return filepath.SkipDir
					}
				}
				return nil
			}

			extension := filepath.Ext(path)
			if extension != ".yaml" && extension != ".yml" {
				return nil
			}

			fContents, err := fs.ReadFile(path)
			if err != nil {
				return err
			}

			if _, err := rf.SliceFromBytes(fContents); err != nil {
				return fmt.Errorf("failed to decode Kubernetes YAML from %s: %w", path, err)
			}
			paths = append(paths, path)
			return nil
		})
		return paths, err
	}

	abs, err := filepath.Abs(dirPath)
	if err != nil {
		return unchangedAction, err
	}

	files, err := scan(abs)
	if err != nil {
		return unchangedAction, err
	}

	kfile := filepath.Join(dirPath, konfig.DefaultKustomizationFileName())
	f, err := fs.Create(kfile)
	if err != nil {
		return unchangedAction, err
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
		resources = append(resources, strings.Replace(file, abs, ".", 1))
	}

	kus.Resources = resources
	kd, err := yaml.Marshal(kus)
	if err != nil {
		// delete the kustomization file
		errf := CleanDirectory(dirPath, createdAction)
		return unchangedAction, fmt.Errorf("%v %v", err, errf)
	}

	return createdAction, os.WriteFile(kfile, kd, os.ModePerm)
}

func adaptSelector(selector *kustomize.Selector) (output *kustypes.Selector) {
	if selector != nil {
		output = &kustypes.Selector{}
		output.Gvk.Group = selector.Group
		output.Gvk.Kind = selector.Kind
		output.Gvk.Version = selector.Version
		output.Name = selector.Name
		output.Namespace = selector.Namespace
		output.LabelSelector = selector.LabelSelector
		output.AnnotationSelector = selector.AnnotationSelector
	}
	return
}

// TODO: remove mutex when kustomize fixes the concurrent map read/write panic
var kustomizeBuildMutex sync.Mutex

// BuildKustomization wraps krusty.MakeKustomizer with the following settings:
// - load files from outside the kustomization.yaml root
// - disable plugins except for the builtin ones
func BuildKustomization(fs filesys.FileSystem, dirPath string) (resmap.ResMap, error) {
	// temporary workaround for concurrent map read and map write bug
	// https://github.com/kubernetes-sigs/kustomize/issues/3659
	kustomizeBuildMutex.Lock()
	defer kustomizeBuildMutex.Unlock()

	buildOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(fs, dirPath)
}

// CleanDirectory removes the kustomization.yaml file from the given directory.
func CleanDirectory(dirPath string, action action) error {
	kfile := filepath.Join(dirPath, konfig.DefaultKustomizationFileName())
	originalFile := filepath.Join(dirPath, originalKustomizationFile)

	// restore old file if it exists
	if _, err := os.Stat(originalFile); err == nil {
		err := os.Rename(originalFile, kfile)
		if err != nil {
			return fmt.Errorf("failed to cleanup repository: %w", err)
		}
	}

	if action == createdAction {
		return os.Remove(kfile)
	}

	return nil
}

// copyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist or else trucnated.
func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}

	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}

	defer func() {
		errf := out.Close()
		if err == nil {
			err = errf
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return
	}

	return
}
