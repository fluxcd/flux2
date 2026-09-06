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

package flags

import (
	"fmt"
	"strings"

	"github.com/fluxcd/pkg/oci"
)

var supportedLayerTypes = []oci.LayerType{oci.LayerTypeTarball, oci.LayerTypeStatic}

type LayerType oci.LayerType

func (l *LayerType) String() string {
	return string(*l)
}

func (l *LayerType) Set(str string) error {
	for _, t := range supportedLayerTypes {
		if oci.LayerType(str) == t {
			*l = LayerType(t)
			return nil
		}
	}
	return fmt.Errorf("unsupported layer type '%s', must be one of: %s", str, l.Type())
}

func (l *LayerType) Type() string {
	types := make([]string, 0, len(supportedLayerTypes))
	for _, t := range supportedLayerTypes {
		types = append(types, string(t))
	}
	return strings.Join(types, "|")
}

func (l *LayerType) Description() string {
	return fmt.Sprintf("how the selected layer should be extracted: '%s' untars the layer to the output directory, '%s' copies the layer to the output file (default: detect from the layer content)",
		oci.LayerTypeTarball, oci.LayerTypeStatic)
}
