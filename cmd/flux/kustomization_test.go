//go:build e2e
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

func TestKustomizationFromGit(t *testing.T) {
	namespace := allocateNamespace("tkfg")
	del, err := execSetupTestNamespace(namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(del)

	tmpl := map[string]string{"ns": namespace}

	cases := []struct {
		args       string
		goldenFile string
		tmpl       map[string]string
	}{
		{
			"create source git tkfg --url=https://github.com/stefanprodan/podinfo --branch=main --tag=6.3.5",
			"testdata/kustomization/create_source_git.golden",
			nil,
		},
		{
			"create kustomization tkfg --source=tkfg --path=./deploy/overlays/dev --prune=true --interval=5m --health-check=Deployment/frontend.dev --health-check=Deployment/backend.dev --health-check-timeout=3m",
			"testdata/kustomization/create_kustomization_from_git.golden",
			nil,
		},
		{
			"get kustomization tkfg",
			"testdata/kustomization/get_kustomization_from_git.golden",
			nil,
		},
		{
			"reconcile kustomization tkfg --with-source",
			"testdata/kustomization/reconcile_kustomization_from_git.golden",
			tmpl,
		},
		{
			"suspend kustomization tkfg",
			"testdata/kustomization/suspend_kustomization_from_git.golden",
			tmpl,
		},
		{
			"suspend kustomization tkfg foo tkfg bar",
			"testdata/kustomization/suspend_kustomization_from_git_multiple_args.golden",
			tmpl,
		},
		{
			"resume kustomization tkfg foo --wait",
			"testdata/kustomization/resume_kustomization_from_git_multiple_args_wait.golden",
			tmpl,
		},
		{
			"resume kustomization tkfg",
			"testdata/kustomization/resume_kustomization_from_git.golden",
			tmpl,
		},
		{
			"resume kustomization tkfg tkfg",
			"testdata/kustomization/resume_kustomization_from_git_multiple_args.golden",
			tmpl,
		},
		{
			"delete kustomization tkfg --silent",
			"testdata/kustomization/delete_kustomization_from_git.golden",
			tmpl,
		},
	}

	for _, tc := range cases {
		cmd := cmdTestCase{
			args:   tc.args + " -n=" + namespace,
			assert: assertGoldenTemplateFile(tc.goldenFile, tc.tmpl),
		}
		cmd.runTestCmd(t)
	}
}
