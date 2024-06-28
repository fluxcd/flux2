/*
Copyright 2024 The Flux authors

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
	"testing"
)

func TestGitLabVisibility_Set(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		expect    string
		expectErr bool
	}{
		{"private", "private", "private", false},
		{"internal", "internal", "internal", false},
		{"public", "public", "public", false},
		{"unsupported", "unsupported", "", true},
		{"default", "", "private", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p GitLabVisibility
			if err := p.Set(tt.str); (err != nil) != tt.expectErr {
				t.Errorf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if str := p.String(); str != tt.expect {
				t.Errorf("Set() = %v, expect %v", str, tt.expect)
			}
		})
	}
}
