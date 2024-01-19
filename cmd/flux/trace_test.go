//go:build unit
// +build unit

package main

import (
	"testing"
	"time"
)

func TestTraceNoArgs(t *testing.T) {
	cmd := cmdTestCase{
		args:   "trace",
		assert: assertError("either `<resource>/<name>` or `<resource> <name>` is required as an argument"),
	}
	cmd.runTestCmd(t)
}

func toLocalTime(t *testing.T, in string) string {
	ts, err := time.Parse(time.RFC3339, in)
	if err != nil {
		t.Fatalf("Error converting golden test time '%s': %v", in, err)
	}
	return ts.Local().String()
}

func TestTrace(t *testing.T) {
	cases := []struct {
		name       string
		args       string
		objectFile string
		goldenFile string
		tmpl       map[string]string
	}{
		{
			"Deployment",
			"trace podinfo --kind deployment --api-version=apps/v1",
			"testdata/trace/deployment.yaml",
			"testdata/trace/deployment.golden",
			map[string]string{
				"ns":                          allocateNamespace("podinfo"),
				"fluxns":                      allocateNamespace("flux-system"),
				"helmReleaseLastReconcile":    toLocalTime(t, "2021-07-16T15:42:20Z"),
				"helmChartLastReconcile":      toLocalTime(t, "2021-07-16T15:32:09Z"),
				"helmRepositoryLastReconcile": toLocalTime(t, "2021-07-11T00:25:46Z"),
			},
		},
		{
			"HelmRelease",
			"trace podinfo --kind HelmRelease --api-version=helm.toolkit.fluxcd.io/v2beta2",
			"testdata/trace/helmrelease.yaml",
			"testdata/trace/helmrelease.golden",
			map[string]string{
				"ns":                         allocateNamespace("podinfo"),
				"fluxns":                     allocateNamespace("flux-system"),
				"kustomizationLastReconcile": toLocalTime(t, "2021-08-01T04:52:56Z"),
				"gitRepositoryLastReconcile": toLocalTime(t, "2021-07-20T00:48:16Z"),
			},
		},
		{
			"HelmRelease from OCI registry",
			"trace podinfo --kind HelmRelease --api-version=helm.toolkit.fluxcd.io/v2beta2",
			"testdata/trace/helmrelease-oci.yaml",
			"testdata/trace/helmrelease-oci.golden",
			map[string]string{
				"ns":                         allocateNamespace("podinfo"),
				"fluxns":                     allocateNamespace("flux-system"),
				"kustomizationLastReconcile": toLocalTime(t, "2021-08-01T04:52:56Z"),
				"ociRepositoryLastReconcile": toLocalTime(t, "2021-07-20T00:48:16Z"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testEnv.CreateObjectFile(tc.objectFile, tc.tmpl, t)
			cmd := cmdTestCase{
				args:   tc.args + " -n=" + tc.tmpl["ns"],
				assert: assertGoldenTemplateFile(tc.goldenFile, tc.tmpl),
			}
			cmd.runTestCmd(t)
		})
	}
}
