//go:build e2e
// +build e2e

/*
Copyright 2022 The Flux authors

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

func TestSourceOCI(t *testing.T) {
	namespace := allocateNamespace("oci-test")
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
			"create source oci thrfg --url=oci://ghcr.io/stefanprodan/manifests/podinfo --tag=6.3.5 --interval 10m",
			"testdata/oci/create_source_oci.golden",
			nil,
		},
		{
			"get source oci thrfg",
			"testdata/oci/get_oci.golden",
			nil,
		},
		{
			"reconcile source oci thrfg",
			"testdata/oci/reconcile_oci.golden",
			tmpl,
		},
		{
			"suspend source oci thrfg",
			"testdata/oci/suspend_oci.golden",
			tmpl,
		},
		{
			"resume source oci thrfg",
			"testdata/oci/resume_oci.golden",
			tmpl,
		},
		{
			"delete source oci thrfg --silent",
			"testdata/oci/delete_oci.golden",
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
