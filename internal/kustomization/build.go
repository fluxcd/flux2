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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fluxcd/flux2/internal/utils"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const mask string = "**SOPS**"

var defaultTimeout = 80 * time.Second

// Builder builds yaml manifests
// It retrieves the kustomization object from the k8s cluster
// and overlays the manifests with the resources specified in the resourcesPath
type Builder struct {
	client        client.WithWatch
	restMapper    meta.RESTMapper
	name          string
	namespace     string
	resourcesPath string
	kustomization *kustomizev1.Kustomization
	timeout       time.Duration
}

type BuilderOptionFunc func(b *Builder) error

func WithTimeout(timeout time.Duration) BuilderOptionFunc {
	return func(b *Builder) error {
		b.timeout = timeout
		return nil
	}
}

// NewBuilder returns a new Builder
// to dp : create functional options
func NewBuilder(kubeconfig string, kubecontext string, namespace, name, resources string, opts ...BuilderOptionFunc) (*Builder, error) {
	kubeClient, err := utils.KubeClient(kubeconfig, kubecontext)
	if err != nil {
		return nil, err
	}

	cfg, err := utils.KubeConfig(kubeconfig, kubecontext)
	if err != nil {
		return nil, err
	}
	restMapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	b := &Builder{
		client:        kubeClient,
		restMapper:    restMapper,
		name:          name,
		namespace:     namespace,
		resourcesPath: resources,
	}

	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	if b.timeout == 0 {
		b.timeout = defaultTimeout
	}

	return b, nil
}

func (b *Builder) getKustomization(ctx context.Context) (*kustomizev1.Kustomization, error) {
	namespacedName := types.NamespacedName{
		Namespace: b.namespace,
		Name:      b.name,
	}

	k := &kustomizev1.Kustomization{}
	err := b.client.Get(ctx, namespacedName, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Build builds the yaml manifests from the kustomization object
// and overlays the manifests with the resources specified in the resourcesPath
// It expects a kustomization.yaml file in the resourcesPath, and it will
// generate a kustomization.yaml file if it doesn't exist
func (b *Builder) Build() ([]byte, error) {
	m, err := b.build()
	if err != nil {
		return nil, err
	}

	resources, err := m.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	return resources, nil
}

func (b *Builder) build() (resmap.ResMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	// Get the kustomization object
	k, err := b.getKustomization(ctx)
	if err != nil {
		return nil, err
	}

	// generate kustomization.yaml if needed
	saved, err := b.generate(*k, b.resourcesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate kustomization.yaml: %w", err)
	}

	// build the kustomization
	m, err := b.do(ctx, *k, b.resourcesPath)
	if err != nil {
		return nil, err
	}

	// make sure secrets are masked
	for _, res := range m.Resources() {
		err := trimSopsData(res)
		if err != nil {
			return nil, err
		}
	}

	// store the kustomization object
	b.kustomization = k

	// overwrite the kustomization.yaml to make sure it's clean
	err = overwrite(saved, b.resourcesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to restore kustomization.yaml: %w", err)
	}

	return m, nil

}

func (b *Builder) generate(kustomization kustomizev1.Kustomization, dirPath string) ([]byte, error) {
	gen := NewGenerator(&kustomizeImpl{kustomization})
	return gen.WriteFile(dirPath)
}

func (b *Builder) do(ctx context.Context, kustomization kustomizev1.Kustomization, dirPath string) (resmap.ResMap, error) {
	fs := filesys.MakeFsOnDisk()
	m, err := buildKustomization(fs, dirPath)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	for _, res := range m.Resources() {
		// run variable substitutions
		if kustomization.Spec.PostBuild != nil {
			outRes, err := substituteVariables(ctx, b.client, &kustomizeImpl{kustomization}, res)
			if err != nil {
				return nil, fmt.Errorf("var substitution failed for '%s': %w", res.GetName(), err)
			}

			if outRes != nil {
				_, err = m.Replace(res)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return m, nil
}

func trimSopsData(res *resource.Resource) error {
	// sopsMess is the base64 encoded mask
	sopsMess := base64.StdEncoding.EncodeToString([]byte(mask))

	if res.GetKind() == "Secret" {
		dataMap := res.GetDataMap()
		for k, v := range dataMap {
			data, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				if _, ok := err.(base64.CorruptInputError); ok {
					return fmt.Errorf("failed to decode secret data: %w", err)
				}
			}

			if bytes.Contains(data, []byte("sops")) && bytes.Contains(data, []byte("ENC[")) {
				dataMap[k] = sopsMess
			}
		}

		res.SetDataMap(dataMap)
	}

	return nil
}

func overwrite(saved []byte, dirPath string) error {
	kfile := filepath.Join(dirPath, konfig.DefaultKustomizationFileName())
	err := os.WriteFile(kfile, saved, 0644)
	if err != nil {
		return fmt.Errorf("failed to overwrite kustomization.yaml: %w", err)
	}
	return nil
}
