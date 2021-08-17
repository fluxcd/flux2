// +build e2e

package main

import "testing"

func TestKustomizationFromGit(t *testing.T) {
	cases := []struct {
		args       string
		goldenFile string
	}{
		{
			"create source git tkfg --url=https://github.com/stefanprodan/podinfo --branch=main --tag=6.0.0",
			"testdata/kustomization/create_source_git.golden",
		},
		{
			"create kustomization tkfg --source=tkfg --path=./deploy/overlays/dev --prune=true --interval=5m --validation=client --health-check=Deployment/frontend.dev --health-check=Deployment/backend.dev --health-check-timeout=3m",
			"testdata/kustomization/create_kustomization_from_git.golden",
		},
		{
			"get kustomization tkfg",
			"testdata/kustomization/get_kustomization_from_git.golden",
		},
		{
			"reconcile kustomization tkfg --with-source",
			"testdata/kustomization/reconcile_kustomization_from_git.golden",
		},
		{
			"suspend kustomization tkfg",
			"testdata/kustomization/suspend_kustomization_from_git.golden",
		},
		{
			"resume kustomization tkfg",
			"testdata/kustomization/resume_kustomization_from_git.golden",
		},
		{
			"delete kustomization tkfg --silent",
			"testdata/kustomization/delete_kustomization_from_git.golden",
		},
	}

	namespace := "tkfg"
	del, err := setupTestNamespace(namespace)
	if err != nil {
		t.Fatal(err)
	}
	defer del()

	for _, tc := range cases {
		cmd := cmdTestCase{
			args:            tc.args + " -n=" + namespace,
			goldenFile:      tc.goldenFile,
			testClusterMode: ExistingClusterMode,
		}
		cmd.runTestCmd(t)
	}
}
