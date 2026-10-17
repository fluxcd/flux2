//go:build unit

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
	"testing"
)

func TestArtifactGeneratorCommands(t *testing.T) {
	tmpl := map[string]string{
		"ns": allocateNamespace("artifact-generator"),
	}
	testEnv.CreateObjectFile("./testdata/artifact_generator/objects.yaml", tmpl, t)

	// The cases are ordered: suspend first, then reconcile on the suspended
	// object, then resume. Resume waits for readiness, which is satisfied by
	// the Ready condition set on the object fixture.
	cases := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "suspend without name",
			args:   "suspend artifact generator -n " + tmpl["ns"],
			assert: assertError("artifactgenerator name is required"),
		},
		{
			name:   "resume without name",
			args:   "resume artifact generator -n " + tmpl["ns"],
			assert: assertError("artifactgenerator name is required"),
		},
		{
			name:   "reconcile without name",
			args:   "reconcile artifact generator -n " + tmpl["ns"],
			assert: assertError("ArtifactGenerator name is required"),
		},
		{
			name:   "suspend missing object",
			args:   "suspend artifact generator missing -n " + tmpl["ns"],
			assert: assertGoldenTemplateFile("testdata/artifact_generator/suspend_missing.golden", tmpl),
		},
		{
			name:   "suspend",
			args:   "suspend artifact generator podinfo-gen -n " + tmpl["ns"],
			assert: assertGoldenTemplateFile("testdata/artifact_generator/suspend.golden", tmpl),
		},
		{
			name:   "reconcile suspended object",
			args:   "reconcile artifact generator podinfo-gen -n " + tmpl["ns"],
			assert: assertError("resource is suspended"),
		},
		{
			name:   "resume",
			args:   "resume artifact generator podinfo-gen --timeout=30s -n " + tmpl["ns"],
			assert: assertGoldenTemplateFile("testdata/artifact_generator/resume.golden", tmpl),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tc.args,
				assert: tc.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}
