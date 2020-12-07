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
	"k8s.io/apimachinery/pkg/runtime"
)

// objectContainer is an interface for a wrapper or alias from which we
// can get a controller-runtime deserialisable value. This is used so
// that you can wrap an API type to give it other useful methods, but
// still use values of the wrapper with `client.Client`, which only
// deals with types that have been added to the schema.
type objectContainer interface {
	AsClientObject() runtime.Object
}

// genericContainer is an objectContainer for any runtime.Object. Use
// this if there are no other methods needed.
type genericContainer struct {
	obj runtime.Object
}

func (c genericContainer) AsClientObject() runtime.Object {
	return c.obj
}
