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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Most commands need one or both of the kind (e.g.,
// `"ImageRepository"`) and a human-palatable name for the kind (e.g.,
// `"image repository"`), to be interpolated into output. It's
// convenient to package these up ahead of time, then the command
// implementation can pick whichever it wants to use.
type apiType struct {
	kind, humanKind string
}

// adapter is an interface for a wrapper or alias from which we can
// get a controller-runtime deserialisable value. This is used so that
// you can wrap an API type to give it other useful methods, but still
// use values of the wrapper with `client.Client`, which only deals
// with types that have been added to the schema.
type adapter interface {
	asClientObject() client.Object
}

// copyable is an interface for a wrapper or alias from which we can
// get a deep copied client.Object, required when you e.g. want to
// calculate a patch.
type copyable interface {
	deepCopyClientObject() client.Object
}

// listAdapater is the analogue to adapter, but for lists; the
// controller runtime distinguishes between methods dealing with
// objects and lists.
type listAdapter interface {
	asClientList() client.ObjectList
	len() int
}

// universalAdapter is an adapter for any client.Object. Use this if
// there are no other methods needed.
type universalAdapter struct {
	obj client.Object
}

func (c universalAdapter) asClientObject() client.Object {
	return c.obj
}

// named is for adapters that have Name and Namespace fields, which
// are sometimes handy to get hold of. ObjectMeta implements these, so
// they shouldn't need any extra work.
type named interface {
	GetName() string
	GetNamespace() string
	GetObjectKind() schema.ObjectKind
	SetName(string)
	SetNamespace(string)
}

func copyName(target, source named) {
	target.SetName(source.GetName())
	target.SetNamespace(source.GetNamespace())
}
