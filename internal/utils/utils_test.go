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

package utils

import "testing"

func TestCompatibleVersion(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		target string
		want   bool
	}{
		{"different major version", "1.1.0", "0.1.0", false},
		{"different minor version", "0.1.0", "0.2.0", false},
		{"same version", "0.1.0", "0.1.0", true},
		{"binary patch version ahead", "0.1.1", "0.1.0", true},
		{"target patch version ahead", "0.1.1", "0.1.2", true},
		{"prerelease binary", "0.0.0-dev.0", "0.1.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompatibleVersion(tt.binary, tt.target); got != tt.want {
				t.Errorf("CompatibleVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
