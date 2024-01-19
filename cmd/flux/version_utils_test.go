//go:build unit
// +build unit

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

package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	cmd := cmdTestCase{
		args:   "--version",
		assert: assertGoldenValue("flux version 0.0.0-dev.0\n"),
	}
	cmd.runTestCmd(t)
}

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		args     string
		expected string
	}{
		{
			args:     "version --client",
			expected: "flux: v0.0.0-dev.0\n",
		},
		{
			args:     "version --client -o json",
			expected: "{\n  \"flux\": \"v0.0.0-dev.0\"\n}\n",
		},
	}

	for _, tt := range tests {
		cmd := cmdTestCase{
			args:   tt.args,
			assert: assertGoldenValue(tt.expected),
		}
		cmd.runTestCmd(t)
	}
}
