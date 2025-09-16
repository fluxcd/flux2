/*
Copyright 2025 The Flux authors

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

	swapi "github.com/fluxcd/source-watcher/api/v2/v1beta1"
)

// swapi.ArtifactGenerator

var artifactGeneratorType = apiType{
	kind:         swapi.ArtifactGeneratorKind,
	humanKind:    "artifactgenerator",
	groupVersion: swapi.GroupVersion,
}

type artifactGeneratorAdapter struct {
	*swapi.ArtifactGenerator
}

func (h artifactGeneratorAdapter) asClientObject() client.Object {
	return h.ArtifactGenerator
}

func (h artifactGeneratorAdapter) deepCopyClientObject() client.Object {
	return h.ArtifactGenerator.DeepCopy()
}

// swapi.ArtifactGeneratorList

type artifactGeneratorListAdapter struct {
	*swapi.ArtifactGeneratorList
}

func (h artifactGeneratorListAdapter) asClientList() client.ObjectList {
	return h.ArtifactGeneratorList
}

func (h artifactGeneratorListAdapter) len() int {
	return len(h.ArtifactGeneratorList.Items)
}
