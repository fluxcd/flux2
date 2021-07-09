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

	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
)

// These are general-purpose adapters for attaching methods to, for
// the various commands. The *List adapters implement len(), since
// it's used in at least a couple of commands.

// imagev1.ImageRepository

var imageRepositoryType = apiType{
	kind:      imagev1.ImageRepositoryKind,
	humanKind: "image repository",
}

type imageRepositoryAdapter struct {
	*imagev1.ImageRepository
}

func (a imageRepositoryAdapter) asClientObject() client.Object {
	return a.ImageRepository
}

func (a imageRepositoryAdapter) deepCopyClientObject() client.Object {
	return a.ImageRepository.DeepCopy()
}

// imagev1.ImageRepositoryList

type imageRepositoryListAdapter struct {
	*imagev1.ImageRepositoryList
}

func (a imageRepositoryListAdapter) asClientList() client.ObjectList {
	return a.ImageRepositoryList
}

func (a imageRepositoryListAdapter) len() int {
	return len(a.ImageRepositoryList.Items)
}

// imagev1.ImagePolicy

var imagePolicyType = apiType{
	kind:      imagev1.ImagePolicyKind,
	humanKind: "image policy",
}

type imagePolicyAdapter struct {
	*imagev1.ImagePolicy
}

func (a imagePolicyAdapter) asClientObject() client.Object {
	return a.ImagePolicy
}

// imagev1.ImagePolicyList

type imagePolicyListAdapter struct {
	*imagev1.ImagePolicyList
}

func (a imagePolicyListAdapter) asClientList() client.ObjectList {
	return a.ImagePolicyList
}

func (a imagePolicyListAdapter) len() int {
	return len(a.ImagePolicyList.Items)
}

// autov1.ImageUpdateAutomation

var imageUpdateAutomationType = apiType{
	kind:      autov1.ImageUpdateAutomationKind,
	humanKind: "image update automation",
}

type imageUpdateAutomationAdapter struct {
	*autov1.ImageUpdateAutomation
}

func (a imageUpdateAutomationAdapter) asClientObject() client.Object {
	return a.ImageUpdateAutomation
}

func (a imageUpdateAutomationAdapter) deepCopyClientObject() client.Object {
	return a.ImageUpdateAutomation.DeepCopy()
}

// autov1.ImageUpdateAutomationList

type imageUpdateAutomationListAdapter struct {
	*autov1.ImageUpdateAutomationList
}

func (a imageUpdateAutomationListAdapter) asClientList() client.ObjectList {
	return a.ImageUpdateAutomationList
}

func (a imageUpdateAutomationListAdapter) len() int {
	return len(a.ImageUpdateAutomationList.Items)
}
