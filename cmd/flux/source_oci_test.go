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
	cases := []struct {
		args       string
		goldenFile string
	}{
		{
			"create source oci thrfg --url=ghcr.io/stefanprodan/manifests/podinfo --tag=6.1.6 --interval 10m",
			"testdata/oci/create_source_oci.golden",
		},
		{
			"get source oci thrfg",
			"testdata/oci/get_oci.golden",
		},
		{
			"reconcile source oci thrfg",
			"testdata/oci/reconcile_oci.golden",
		},
		{
			"suspend source oci thrfg",
			"testdata/oci/suspend_oci.golden",
		},
		{
			"resume source oci thrfg",
			"testdata/oci/resume_oci.golden",
		},
		{
			"delete source oci thrfg --silent",
			"testdata/oci/delete_oci.golden",
		},
	}

	namespace := allocateNamespace("oci-test")
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
