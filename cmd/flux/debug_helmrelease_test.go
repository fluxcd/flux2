//go:build unit
// +build unit

package main

import (
	"testing"
)

func TestDebugHelmRelease(t *testing.T) {
	namespace := allocateNamespace("debug")

	objectFile := "testdata/debug_helmrelease/objects.yaml"
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
			"debug status",
			"debug helmrelease test-values-inline --show-status --show-values=false",
			"testdata/debug_helmrelease/status.golden.yaml",
			tmpl,
		},
		{
			"debug values",
			"debug helmrelease test-values-inline --show-values --show-status=false",
			"testdata/debug_helmrelease/values-inline.golden.yaml",
			tmpl,
		},
		{
			"debug values from",
			"debug helmrelease test-values-from --show-values --show-status=false",
			"testdata/debug_helmrelease/values-from.golden.yaml",
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
