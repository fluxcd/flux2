/*
Copyright 2020 The Flux authors

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
	"sigs.k8s.io/controller-runtime/pkg/client"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
)

// kustomizev1.Kustomization

var kustomizationType = apiType{
	kind:      kustomizev1.KustomizationKind,
	humanKind: "kustomizations",
}

type kustomizationAdapter struct {
	*kustomizev1.Kustomization
}

func (a kustomizationAdapter) asClientObject() client.Object {
	return a.Kustomization
}

func (a kustomizationAdapter) deepCopyClientObject() client.Object {
	return a.Kustomization.DeepCopy()
}

// kustomizev1.KustomizationList

type kustomizationListAdapter struct {
	*kustomizev1.KustomizationList
}

func (a kustomizationListAdapter) asClientList() client.ObjectList {
	return a.KustomizationList
}

func (a kustomizationListAdapter) len() int {
	return len(a.KustomizationList.Items)
}
