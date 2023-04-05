//go:build unit
// +build unit

/*
Copyright 2023 The Flux authors

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

import "testing"

func Test_GetCmd(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile("./testdata/get/objects.yaml", tmpl, t)

	tests := []struct {
		name     string
		args     string
		expected string
	}{
		{
			name:     "no label selector",
			expected: "testdata/get/get.golden",
		},
		{
			name:     "equal label selector",
			args:     "-l sharding.fluxcd.io/key=shard1",
			expected: "testdata/get/get_label_one.golden",
		},
		{
			name:     "notin label selector",
			args:     `-l "sharding.fluxcd.io/key notin (shard1, shard2)"`,
			expected: "testdata/get/get_label_two.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   "get sources git " + tt.args + " -n " + tmpl["fluxns"],
				assert: assertGoldenTemplateFile(tt.expected, nil),
			}

			cmd.runTestCmd(t)
		})
	}
}
