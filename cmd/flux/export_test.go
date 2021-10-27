// +build unit

package main

import (
	"testing"
)

func TestExport(t *testing.T) {
	cases := []struct {
		name       string
		arg        string
		goldenFile string
	}{
		{
			"alert-provider",
			"export alert-provider slack",
			"testdata/export/provider.yaml",
		},
		{
			"alert",
			"export alert flux-system",
			"testdata/export/alert.yaml",
		},
		{
			"image policy",
			"export image policy flux-system",
			"testdata/export/image-policy.yaml",
		},
		{
			"image repository",
			"export image repository flux-system",
			"testdata/export/image-repo.yaml",
		},
		{
			"image update",
			"export image update flux-system",
			"testdata/export/image-update.yaml",
		},
		{
			"source git",
			"export source git flux-system",
			"testdata/export/git-repo.yaml",
		},
		{
			"source helm",
			"export source helm flux-system",
			"testdata/export/helm-repo.yaml",
		},
		{
			"receiver",
			"export receiver flux-system",
			"testdata/export/receiver.yaml",
		},
		{
			"kustomization",
			"export kustomization flux-system",
			"testdata/export/ks.yaml",
		},
		{
			"helmrelease",
			"export helmrelease flux-system",
			"testdata/export/helm-release.yaml",
		},
		{
			"bucket",
			"export source bucket flux-system",
			"testdata/export/bucket.yaml",
		},
	}

	objectFile := "testdata/export/objects.yaml"
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	testEnv.CreateObjectFile(objectFile, tmpl, t)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.arg + " -n=" + tmpl["fluxns"],
				assert: assertGoldenTemplateFile(tt.goldenFile, tmpl),
			}

			cmd.runTestCmd(t)
		})
	}
}
