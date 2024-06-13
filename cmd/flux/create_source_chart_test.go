//go:build unit
// +build unit

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

package main

import "testing"

func TestCreateSourceChart(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupSourceChart(t, tmpl)

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "missing name",
			args:   "create source chart --export",
			assert: assertError("name is required"),
		},
		{
			name:   "missing source reference",
			args:   "create source chart podinfo --export ",
			assert: assertError("chart source is required"),
		},
		{
			name:   "missing chart name",
			args:   "create source chart podinfo --source helmrepository/podinfo --export",
			assert: assertError("chart name or path is required"),
		},
		{
			name:   "unknown source kind",
			args:   "create source chart podinfo --source foobar/podinfo --export",
			assert: assertError(`invalid argument "foobar/podinfo" for "--source" flag: source kind 'foobar' is not supported, must be one of: HelmRepository, GitRepository, Bucket`),
		},
		{
			name:   "basic chart",
			args:   "create source chart podinfo --source helmrepository/podinfo --chart podinfo --export",
			assert: assertGoldenTemplateFile("testdata/create_source_chart/basic.yaml", tmpl),
		},
		{
			name:   "chart with basic signature verification",
			args:   "create source chart podinfo --source helmrepository/podinfo --chart podinfo --verify-provider cosign --export",
			assert: assertGoldenTemplateFile("testdata/create_source_chart/verify_basic.yaml", tmpl),
		},
		{
			name:   "unknown signature verification provider",
			args:   "create source chart podinfo --source helmrepository/podinfo --chart podinfo --verify-provider foobar --export",
			assert: assertError(`invalid argument "foobar" for "--verify-provider" flag: source OCI verify provider 'foobar' is not supported, must be one of: cosign`),
		},
		{
			name:   "chart with complete signature verification",
			args:   "create source chart podinfo --source helmrepository/podinfo --chart podinfo --verify-provider cosign --verify-issuer foo --verify-subject bar --export",
			assert: assertGoldenTemplateFile("testdata/create_source_chart/verify_complete.yaml", tmpl),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}

func setupSourceChart(t *testing.T, tmpl map[string]string) {
	t.Helper()
	testEnv.CreateObjectFile("./testdata/create_source_chart/setup-source.yaml", tmpl, t)
}
