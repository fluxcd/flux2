//go:build unit
// +build unit

package main

import (
	"testing"
)

func TestTraceNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:   "trace",
		assert: assertError("either `<resource>/<name>` or `<resource> <name>` is required as an argument"),
	}
	cmd.runTestCmd(t)
}

func TestTrace(t *testing.T) {
	cases := []struct {
		name       string
		args       string
		objectFile string
		goldenFile string
	}{
		{
			"Deployment",
			"trace podinfo --kind deployment --api-version=apps/v1",
			"testdata/trace/deployment.yaml",
			"testdata/trace/deployment.golden",
		},
		{
			"HelmRelease",
			"trace podinfo --kind HelmRelease --api-version=helm.toolkit.fluxcd.io/v2beta1",
			"testdata/trace/helmrelease.yaml",
			"testdata/trace/helmrelease.golden",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := map[string]string{
				"ns":     allocateNamespace("podinfo"),
				"fluxns": allocateNamespace("flux-system"),
			}
			testEnv.CreateObjectFile(tc.objectFile, tmpl, t)
			cmd := cmdTestCase{
				args:   tc.args + " -n=" + tmpl["ns"],
				assert: assertGoldenTemplateFile(tc.goldenFile, tmpl),
			}
			cmd.runTestCmd(t)
		})
	}
}
