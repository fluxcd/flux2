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
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

// helmv2.HelmRelease

var helmReleaseType = apiType{
	kind:      helmv2.HelmReleaseKind,
	humanKind: "helmreleases",
}

type helmReleaseAdapter struct {
	*helmv2.HelmRelease
}

func (h helmReleaseAdapter) asClientObject() client.Object {
	return h.HelmRelease
}

func (h helmReleaseAdapter) deepCopyClientObject() client.Object {
	return h.HelmRelease.DeepCopy()
}

// helmv2.HelmReleaseList

type helmReleaseListAdapter struct {
	*helmv2.HelmReleaseList
}

func (h helmReleaseListAdapter) asClientList() client.ObjectList {
	return h.HelmReleaseList
}

func (h helmReleaseListAdapter) len() int {
	return len(h.HelmReleaseList.Items)
}
