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

import (
	"testing"
)

func TestExport(t *testing.T) {
	namespace := allocateNamespace("flux-system")

	objectFile := "testdata/export/objects.yaml"
	tmpl := map[string]string{
		"fluxns": namespace,
	}
	testEnv.CreateObjectFile(objectFile, tmpl, t)

	cases := []struct {
		name       string
		arg        string
		goldenFile string
		tmpl       map[string]string
	}{
		{
			"alert-provider",
			"export alert-provider slack",
			"testdata/export/provider.yaml",
			tmpl,
		},
		{
			"alert",
			"export alert flux-system",
			"testdata/export/alert.yaml",
			tmpl,
		},
		{
			"image policy",
			"export image policy flux-system",
			"testdata/export/image-policy.yaml",
			tmpl,
		},
		{
			"image repository",
			"export image repository flux-system",
			"testdata/export/image-repo.yaml",
			tmpl,
		},
		{
			"image update",
			"export image update flux-system",
			"testdata/export/image-update.yaml",
			tmpl,
		},
		{
			"source git",
			"export source git flux-system",
			"testdata/export/git-repo.yaml",
			tmpl,
		},
		{
			"source chart",
			"export source chart flux-system",
			"testdata/export/helm-chart.yaml",
			tmpl,
		},
		{
			"source helm",
			"export source helm flux-system",
			"testdata/export/helm-repo.yaml",
			tmpl,
		},
		{
			"receiver",
			"export receiver flux-system",
			"testdata/export/receiver.yaml",
			tmpl,
		},
		{
			"kustomization",
			"export kustomization flux-system",
			"testdata/export/ks.yaml",
			tmpl,
		},
		{
			"helmrelease",
			"export helmrelease flux-system",
			"testdata/export/helm-release.yaml",
			tmpl,
		},
		{
			"bucket",
			"export source bucket flux-system",
			"testdata/export/bucket.yaml",
			tmpl,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.arg + " -n=" + namespace,
				assert: assertGoldenTemplateFile(tt.goldenFile, tmpl),
			}

			cmd.runTestCmd(t)
		})
	}
}
