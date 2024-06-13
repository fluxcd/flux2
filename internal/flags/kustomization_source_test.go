//go:build !e2e
// +build !e2e

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

package flags

import (
	"fmt"
	"testing"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
)

func TestKustomizationSource_Set(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		expect    string
		expectErr bool
	}{
		{"supported", fmt.Sprintf("%s/foo", sourcev1.GitRepositoryKind), fmt.Sprintf("%s/foo", sourcev1.GitRepositoryKind), false},
		{"default kind", "foo", fmt.Sprintf("%s/foo", sourcev1.GitRepositoryKind), false},
		{"lower case kind", "gitrepository/foo", fmt.Sprintf("%s/foo", sourcev1.GitRepositoryKind), false},
		{"unsupported", "Unsupported/kind", "", true},
		{"missing name", fmt.Sprintf("%s/", sourcev1.GitRepositoryKind), "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s KustomizationSource
			if err := s.Set(tt.str); (err != nil) != tt.expectErr {
				t.Errorf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if str := s.String(); str != tt.expect {
				t.Errorf("Set() = %v, expect %v", str, tt.expect)
			}
		})
	}
}
