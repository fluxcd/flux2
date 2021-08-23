// +build e2e

package main

import "testing"

func TestHelmReleaseFromGit(t *testing.T) {
	cases := []struct {
		args       string
		goldenFile string
	}{
		{
			"create source git thrfg --url=https://github.com/stefanprodan/podinfo --branch=main --tag=6.0.0",
			"testdata/helmrelease/create_source_git.golden",
		},
		{
			"create helmrelease thrfg --source=GitRepository/thrfg --chart=./charts/podinfo",
			"testdata/helmrelease/create_helmrelease_from_git.golden",
		},
		{
			"get helmrelease thrfg",
			"testdata/helmrelease/get_helmrelease_from_git.golden",
		},
		{
			"reconcile helmrelease thrfg --with-source",
			"testdata/helmrelease/reconcile_helmrelease_from_git.golden",
		},
		{
			"suspend helmrelease thrfg",
			"testdata/helmrelease/suspend_helmrelease_from_git.golden",
		},
		{
			"resume helmrelease thrfg",
			"testdata/helmrelease/resume_helmrelease_from_git.golden",
		},
		{
			"delete helmrelease thrfg --silent",
			"testdata/helmrelease/delete_helmrelease_from_git.golden",
		},
	}

	namespace := "thrfg"
	del, err := setupTestNamespace(namespace)
	if err != nil {
		t.Fatal(err)
	}
	defer del()

	for _, tc := range cases {
		cmd := cmdTestCase{
			args:            tc.args + " -n=" + namespace,
			assert:          assertGoldenFile(tc.goldenFile),
			testClusterMode: ExistingClusterMode,
		}
		cmd.runTestCmd(t)
	}
}
