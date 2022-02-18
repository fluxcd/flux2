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

package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fluxcd/pkg/ssa"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"

	"github.com/fluxcd/flux2/pkg/manifestgen/kustomization"
)

// Apply is the equivalent of 'kubectl apply --server-side -f'.
// If the given manifest is a kustomization.yaml, then apply performs the equivalent of 'kubectl apply --server-side -k'.
func Apply(ctx context.Context, rcg genericclioptions.RESTClientGetter, manifestPath string) (string, error) {
	objs, err := readObjects(manifestPath)
	if err != nil {
		return "", err
	}

	if len(objs) == 0 {
		return "", fmt.Errorf("no Kubernetes objects found at: %s", manifestPath)
	}

	if err := ssa.SetNativeKindsDefaults(objs); err != nil {
		return "", err
	}

	changeSet := ssa.NewChangeSet()

	// contains only CRDs and Namespaces
	var stageOne []*unstructured.Unstructured

	// contains all objects except for CRDs and Namespaces
	var stageTwo []*unstructured.Unstructured

	for _, u := range objs {
		if ssa.IsClusterDefinition(u) {
			stageOne = append(stageOne, u)
		} else {
			stageTwo = append(stageTwo, u)
		}
	}

	if len(stageOne) > 0 {
		cs, err := applySet(ctx, rcg, stageOne)
		if err != nil {
			return "", err
		}
		changeSet.Append(cs.Entries)
	}

	if err := waitForSet(rcg, changeSet); err != nil {
		return "", err
	}

	if len(stageTwo) > 0 {
		cs, err := applySet(ctx, rcg, stageTwo)
		if err != nil {
			return "", err
		}
		changeSet.Append(cs.Entries)
	}

	return changeSet.String(), nil
}

func readObjects(manifestPath string) ([]*unstructured.Unstructured, error) {
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, err
	}

	if filepath.Base(manifestPath) == konfig.DefaultKustomizationFileName() {
		resources, err := kustomization.Build(filepath.Dir(manifestPath))
		if err != nil {
			return nil, err
		}
		return ssa.ReadObjects(bytes.NewReader(resources))
	}

	ms, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer ms.Close()

	return ssa.ReadObjects(bufio.NewReader(ms))
}

func newManager(rcg genericclioptions.RESTClientGetter) (*ssa.ResourceManager, error) {
	cfg, err := KubeConfig(rcg)
	if err != nil {
		return nil, err
	}
	restMapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(cfg, client.Options{Mapper: restMapper, Scheme: NewScheme()})
	if err != nil {
		return nil, err
	}
	kubePoller := polling.NewStatusPoller(kubeClient, restMapper, polling.Options{})

	return ssa.NewResourceManager(kubeClient, kubePoller, ssa.Owner{
		Field: "flux",
		Group: "fluxcd.io",
	}), nil

}

func applySet(ctx context.Context, rcg genericclioptions.RESTClientGetter, objects []*unstructured.Unstructured) (*ssa.ChangeSet, error) {
	man, err := newManager(rcg)
	if err != nil {
		return nil, err
	}

	return man.ApplyAll(ctx, objects, ssa.DefaultApplyOptions())
}

func waitForSet(rcg genericclioptions.RESTClientGetter, changeSet *ssa.ChangeSet) error {
	man, err := newManager(rcg)
	if err != nil {
		return err
	}
	return man.WaitForSet(changeSet.ToObjMetadataSet(), ssa.WaitOptions{Interval: 2 * time.Second, Timeout: time.Minute})
}
