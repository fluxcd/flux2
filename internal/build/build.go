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

package build

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/theckman/yacspin"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/kustomize"
	runclient "github.com/fluxcd/pkg/runtime/client"

	"github.com/fluxcd/flux2/internal/utils"
)

const (
	controllerName      = "kustomize-controller"
	controllerGroup     = "kustomize.toolkit.fluxcd.io"
	mask                = "**SOPS**"
	dockercfgSecretType = "kubernetes.io/dockerconfigjson"
	typeField           = "type"
	dataField           = "data"
	stringDataField     = "stringData"
)

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
	// mu is used to synchronize access to the kustomization file
	mu            sync.Mutex
	action        kustomize.Action
	kustomization *kustomizev1.Kustomization
	timeout       time.Duration
	spinner       *yacspin.Spinner
}

type BuilderOptionFunc func(b *Builder) error

func WithTimeout(timeout time.Duration) BuilderOptionFunc {
	return func(b *Builder) error {
		b.timeout = timeout
		return nil
	}
}

func WithProgressBar() BuilderOptionFunc {
	return func(b *Builder) error {
		// Add a spiner
		cfg := yacspin.Config{
			Frequency:       100 * time.Millisecond,
			CharSet:         yacspin.CharSets[59],
			Suffix:          "Kustomization diffing...",
			SuffixAutoColon: true,
			Message:         "running dry-run",
			StopCharacter:   "✓",
			StopColors:      []string{"fgGreen"},
		}
		spinner, err := yacspin.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to create spinner: %w", err)
		}
		b.spinner = spinner

		return nil
	}
}

// NewBuilder returns a new Builder
// to dp : create functional options
func NewBuilder(rcg *genericclioptions.ConfigFlags, clientOpts *runclient.Options, name, resources string, opts ...BuilderOptionFunc) (*Builder, error) {
	kubeClient, err := utils.KubeClient(rcg, clientOpts)
	if err != nil {
		return nil, err
	}

	restMapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	b := &Builder{
		client:        kubeClient,
		restMapper:    restMapper,
		name:          name,
		namespace:     *rcg.Namespace,
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

func (b *Builder) build() (m resmap.ResMap, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	// Get the kustomization object
	k, err := b.getKustomization(ctx)
	if err != nil {
		return
	}

	// store the kustomization object
	b.kustomization = k

	// generate kustomization.yaml if needed
	action, er := b.generate(*k, b.resourcesPath)
	if er != nil {
		errf := kustomize.CleanDirectory(b.resourcesPath, action)
		err = fmt.Errorf("failed to generate kustomization.yaml: %w", fmt.Errorf("%v %v", er, errf))
		return
	}

	b.action = action

	defer func() {
		errf := b.Cancel()
		if err == nil {
			err = errf
		}
	}()

	// build the kustomization
	m, err = b.do(ctx, *k, b.resourcesPath)
	if err != nil {
		return
	}

	for _, res := range m.Resources() {
		// set owner labels
		err = b.setOwnerLabels(res)
		if err != nil {
			return
		}

		// make sure secrets are masked
		err = maskSopsData(res)
		if err != nil {
			return
		}
	}

	return

}

func (b *Builder) generate(kustomization kustomizev1.Kustomization, dirPath string) (kustomize.Action, error) {
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kustomization)
	if err != nil {
		return "", err
	}
	gen := kustomize.NewGenerator(unstructured.Unstructured{Object: data})

	// acuire the lock
	b.mu.Lock()
	defer b.mu.Unlock()

	return gen.WriteFile(dirPath, kustomize.WithSaveOriginalKustomization())
}

func (b *Builder) do(ctx context.Context, kustomization kustomizev1.Kustomization, dirPath string) (resmap.ResMap, error) {
	fs := filesys.MakeFsOnDisk()

	// acuire the lock
	b.mu.Lock()
	defer b.mu.Unlock()

	m, err := kustomize.BuildKustomization(fs, dirPath)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	for _, res := range m.Resources() {
		// run variable substitutions
		if kustomization.Spec.PostBuild != nil {
			data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kustomization)
			if err != nil {
				return nil, err
			}
			outRes, err := kustomize.SubstituteVariables(ctx, b.client, unstructured.Unstructured{Object: data}, res)
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

func (b *Builder) setOwnerLabels(res *resource.Resource) error {
	labels := res.GetLabels()

	labels[controllerGroup+"/name"] = b.kustomization.GetName()
	labels[controllerGroup+"/namespace"] = b.kustomization.GetNamespace()

	err := res.SetLabels(labels)
	if err != nil {
		return err
	}

	return nil
}

func maskSopsData(res *resource.Resource) error {
	// sopsMess is the base64 encoded mask
	sopsMess := base64.StdEncoding.EncodeToString([]byte(mask))

	if res.GetKind() == "Secret" {
		// get both data and stringdata maps as a secret can have both
		dataMap := res.GetDataMap()
		stringDataMap := getStringDataMap(res)
		asYaml, err := res.AsYAML()
		if err != nil {
			return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
		}

		// delete any sops data as we don't want to expose it
		// assume that both data and stringdata are encrypted
		if bytes.Contains(asYaml, []byte("sops:")) && bytes.Contains(asYaml, []byte("mac: ENC[")) {
			// delete the sops object
			res.PipeE(yaml.FieldClearer{Name: "sops"})

			secretType, err := res.GetFieldValue(typeField)
			if err != nil {
				return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
			}

			if v, ok := secretType.(string); ok && v == dockercfgSecretType {
				// if the secret is a json docker config secret, we need to mask the data with a json object
				err := maskDockerconfigjsonSopsData(dataMap, true)
				if err != nil {
					return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
				}

				err = maskDockerconfigjsonSopsData(stringDataMap, false)
				if err != nil {
					return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
				}

			} else {
				for k := range dataMap {
					dataMap[k] = sopsMess
				}

				for k := range stringDataMap {
					stringDataMap[k] = mask
				}
			}
		} else {
			err := maskBase64EncryptedSopsData(dataMap, sopsMess)
			if err != nil {
				return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
			}

			err = maskSopsDataInStringDataSecret(stringDataMap, sopsMess)
			if err != nil {
				return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
			}
		}

		// set the data and stringdata maps
		res.SetDataMap(dataMap)

		if len(stringDataMap) > 0 {
			err = res.SetMapField(yaml.NewMapRNode(&stringDataMap), stringDataField)
			if err != nil {
				return fmt.Errorf("failed to mask secret %s sops data: %w", res.GetName(), err)
			}
		}
	}

	return nil
}

func getStringDataMap(rn *resource.Resource) map[string]string {
	n, err := rn.Pipe(yaml.Lookup(stringDataField))
	if err != nil {
		return nil
	}
	result := map[string]string{}
	_ = n.VisitFields(func(node *yaml.MapNode) error {
		result[yaml.GetValue(node.Key)] = yaml.GetValue(node.Value)
		return nil
	})
	return result
}

func maskDockerconfigjsonSopsData(dataMap map[string]string, encode bool) error {
	sopsMess := struct {
		Mask string `json:"mask"`
	}{
		Mask: mask,
	}

	maskJson, err := json.Marshal(sopsMess)
	if err != nil {
		return err
	}

	if encode {
		for k := range dataMap {
			dataMap[k] = base64.StdEncoding.EncodeToString(maskJson)
		}
		return nil
	}

	for k := range dataMap {
		dataMap[k] = string(maskJson)
	}

	return nil
}

func maskBase64EncryptedSopsData(dataMap map[string]string, mask string) error {
	for k, v := range dataMap {
		data, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			if _, ok := err.(base64.CorruptInputError); ok {
				return err
			}
		}

		if bytes.Contains(data, []byte("sops")) && bytes.Contains(data, []byte("ENC[")) {
			dataMap[k] = mask
		}
	}

	return nil
}

func maskSopsDataInStringDataSecret(stringDataMap map[string]string, mask string) error {
	for k, v := range stringDataMap {
		if bytes.Contains([]byte(v), []byte("sops")) && bytes.Contains([]byte(v), []byte("ENC[")) {
			stringDataMap[k] = mask
		}
	}

	return nil
}

// Cancel cancels the build
// It restores a clean reprository
func (b *Builder) Cancel() error {
	// acuire the lock
	b.mu.Lock()
	defer b.mu.Unlock()

	err := kustomize.CleanDirectory(b.resourcesPath, b.action)
	if err != nil {
		return err
	}

	return nil
}
