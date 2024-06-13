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
	"github.com/fluxcd/cli-utils/pkg/object"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// statusable is used to see if a resource is considered ready in the usual way
type statusable interface {
	adapter
	// this is implemented by ObjectMeta
	GetGeneration() int64
	getObservedGeneration() int64
}

// oldConditions represents the deprecated API which is sunsetting.
type oldConditions interface {
	// this is usually implemented by GOTK API objects because it's used by pkg/apis/meta
	GetStatusConditions() *[]metav1.Condition
}

func buildComponentObjectRefs(components ...string) ([]object.ObjMetadata, error) {
	var objRefs []object.ObjMetadata
	for _, deployment := range components {
		objRefs = append(objRefs, object.ObjMetadata{
			Namespace: *kubeconfigArgs.Namespace,
			Name:      deployment,
			GroupKind: schema.GroupKind{Group: "apps", Kind: "Deployment"},
		})
	}
	return objRefs, nil
}
