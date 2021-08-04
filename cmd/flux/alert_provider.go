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

	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
)

// notificationv1.Provider

var alertProviderType = apiType{
	kind:      notificationv1.ProviderKind,
	humanKind: "alert provider",
}

type alertProviderAdapter struct {
	*notificationv1.Provider
}

func (a alertProviderAdapter) asClientObject() client.Object {
	return a.Provider
}

func (a alertProviderAdapter) deepCopyClientObject() client.Object {
	return a.Provider.DeepCopy()
}

// notificationv1.Provider

type alertProviderListAdapter struct {
	*notificationv1.ProviderList
}

func (a alertProviderListAdapter) asClientList() client.ObjectList {
	return a.ProviderList
}

func (a alertProviderListAdapter) len() int {
	return len(a.ProviderList.Items)
}
