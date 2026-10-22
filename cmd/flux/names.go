/*
Copyright 2026 The Flux authors

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
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// uniqueNames returns names without duplicates, keeping the order of
// first occurrence.
func uniqueNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// resetObject sets the value obj points to back to its zero value, so
// that fields from a previous Get do not leak into the next one.
func resetObject(obj client.Object) {
	v := reflect.ValueOf(obj).Elem()
	v.Set(reflect.Zero(v.Type()))
}
