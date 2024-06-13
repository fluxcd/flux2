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
	"testing"
)

func TestRelativePath_Set(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		expect    string
		expectErr bool
	}{
		{"relative path", "./foo", "./foo", false},
		{"relative path", "foo", "./foo", false},
		{"traversing relative path", "./foo/../bar", "./bar", false},
		{"absolute path", "/foo", "./foo", false},
		{"traversing absolute path", "/foo/../bar", "./bar", false},
		{"traversing overflowing absolute path", "/foo/../../../bar", "./bar", false},
		{"empty", "", "./", false},
		{"relative empty path", "./", "./", false},
		{"double relative empty path", "././", "./", false},
		{"dot path", ".foo", "./.foo", false},
		{"relative dot path", "./.foo", "./.foo", false},
		{"current directory", ".", "./", false},
		{"parent directory", "..", "./", false},
		{"parent directory more qualified", "./..", "./", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p SafeRelativePath
			if err := p.Set(tt.str); (err != nil) != tt.expectErr {
				t.Errorf("Set() error = %v, expectErr %v", err, tt.expectErr)
			}
			if str := p.String(); str != tt.expect {
				t.Errorf("Set() = %v, expect %v", str, tt.expect)
			}
		})
	}
}
