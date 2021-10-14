// +build e2e

/*
Copyright 2021 The Flux authors

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

	namespace := allocateNamespace("thrfg")
	del, err := setupTestNamespace(namespace)
	if err != nil {
		t.Fatal(err)
	}
	defer del()

	for _, tc := range cases {
		cmd := cmdTestCase{
			args:   tc.args + " -n=" + namespace,
			assert: assertGoldenTemplateFile(tc.goldenFile, map[string]string{"ns": namespace}),
		}
		cmd.runTestCmd(t)
	}
}
