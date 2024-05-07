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

func TestCreateHelmRelease(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupHRSource(t, tmpl)

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "missing name",
			args:   "create helmrelease --export",
			assert: assertError("name is required"),
		},
		{
			name:   "missing chart template and chartRef",
			args:   "create helmrelease podinfo --export",
			assert: assertError("chart or chart-ref is required"),
		},
		{
			name:   "unknown source kind",
			args:   "create helmrelease podinfo --source foobar/podinfo --chart podinfo --export",
			assert: assertError(`invalid argument "foobar/podinfo" for "--source" flag: source kind 'foobar' is not supported, must be one of: HelmRepository, GitRepository, Bucket`),
		},
		{
			name:   "unknown chart reference kind",
			args:   "create helmrelease podinfo --chart-ref foobar/podinfo --export",
			assert: assertError(`chart reference kind 'foobar' is not supported, must be one of: OCIRepository, HelmChart`),
		},
		{
			name:   "basic helmrelease",
			args:   "create helmrelease podinfo --source Helmrepository/podinfo --chart podinfo --interval=1m0s --export",
			assert: assertGoldenTemplateFile("testdata/create_hr/basic.yaml", tmpl),
		},
		{
			name:   "chart with OCIRepository source",
			args:   "create helmrelease podinfo --chart-ref OCIRepository/podinfo --interval=1m0s --export",
			assert: assertGoldenTemplateFile("testdata/create_hr/or_basic.yaml", tmpl),
		},
		{
			name:   "chart with HelmChart source",
			args:   "create helmrelease podinfo --chart-ref HelmChart/podinfo --interval=1m0s --export",
			assert: assertGoldenTemplateFile("testdata/create_hr/hc_basic.yaml", tmpl),
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

func setupHRSource(t *testing.T, tmpl map[string]string) {
	t.Helper()
	testEnv.CreateObjectFile("./testdata/create_hr/setup-source.yaml", tmpl, t)
}
