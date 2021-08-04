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

// notificationv1.Alert

var alertType = apiType{
	kind:      notificationv1.AlertKind,
	humanKind: "alert",
}

type alertAdapter struct {
	*notificationv1.Alert
}

func (a alertAdapter) asClientObject() client.Object {
	return a.Alert
}

func (a alertAdapter) deepCopyClientObject() client.Object {
	return a.Alert.DeepCopy()
}

// notificationv1.Alert

type alertListAdapter struct {
	*notificationv1.AlertList
}

func (a alertListAdapter) asClientList() client.ObjectList {
	return a.AlertList
}

func (a alertListAdapter) len() int {
	return len(a.AlertList.Items)
}
