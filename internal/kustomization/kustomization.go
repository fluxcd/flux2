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
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/kustomize"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Kustomize defines the methods to retrieve the kustomization information
// TO DO @souleb: move this to fluxcd/pkg along with generator and varsub
type Kustomize interface {
	client.Object
	GetTargetNamespace() string
	GetPatches() []kustomize.Patch
	GetPatchesStrategicMerge() []apiextensionsv1.JSON
	GetPatchesJSON6902() []kustomize.JSON6902Patch
	GetImages() []kustomize.Image
	GetSubstituteFrom() []SubstituteReference
	GetSubstitute() map[string]string
}

// SubstituteReference contains a reference to a resource containing
// the variables name and value.
type SubstituteReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// TO DO @souleb: this is a temporary hack to get the kustomize object
// from the kustomize controller.
// At some point we should remove this and have the kustomize controller implement
// the Kustomize interface.
type kustomizeImpl struct {
	kustomizev1.Kustomization
}

func (k *kustomizeImpl) GetTargetNamespace() string {
	return k.Spec.TargetNamespace
}

func (k *kustomizeImpl) GetPatches() []kustomize.Patch {
	return k.Spec.Patches
}

func (k *kustomizeImpl) GetPatchesStrategicMerge() []apiextensionsv1.JSON {
	return k.Spec.PatchesStrategicMerge
}

func (k *kustomizeImpl) GetPatchesJSON6902() []kustomize.JSON6902Patch {
	return k.Spec.PatchesJSON6902
}

func (k *kustomizeImpl) GetImages() []kustomize.Image {
	return k.Spec.Images
}

func (k *kustomizeImpl) GetSubstituteFrom() []SubstituteReference {
	refs := make([]SubstituteReference, 0, len(k.Spec.PostBuild.SubstituteFrom))
	for _, s := range k.Spec.PostBuild.SubstituteFrom {
		refs = append(refs, SubstituteReference(s))
	}
	return refs
}

func (k *kustomizeImpl) GetSubstitute() map[string]string {
	return k.Spec.PostBuild.Substitute
}
