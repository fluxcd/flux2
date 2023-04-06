//go:build !e2e
// +build !e2e

/*
Copyright 2020 The Flux CD contributors.

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

package sync

import (
	"fmt"
	"strings"
	"testing"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestGenerate(t *testing.T) {
	opts := MakeDefaultOptions()
	output, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, apiVersion := range []string{sourcev1.GroupVersion.String(), kustomizev1.GroupVersion.String()} {
		if !strings.Contains(output.Content, apiVersion) {
			t.Errorf("apiVersion '%s' not found", apiVersion)
		}
	}

	fmt.Println(output.Content)
}
