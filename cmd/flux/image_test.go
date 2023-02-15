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

import "testing"

func TestImageScanning(t *testing.T) {
	namespace := allocateNamespace("tis")
	del, err := execSetupTestNamespace(namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(del)

	cases := []struct {
		args       string
		goldenFile string
	}{
		{
			"create image repository podinfo --image=ghcr.io/stefanprodan/podinfo --interval=10m",
			"testdata/image/create_image_repository.golden",
		},
		{
			"create image policy podinfo-semver --image-ref=podinfo --interval=10m --select-semver=5.0.x",
			"testdata/image/create_image_policy.golden",
		},
		{
			"get image policy podinfo-semver",
			"testdata/image/get_image_policy_semver.golden",
		},
		{
			`create image policy podinfo-regex --image-ref=podinfo --interval=10m --select-semver=">4.0.0" --filter-regex="5\.0\.0"`,
			"testdata/image/create_image_policy.golden",
		},
		{
			"get image policy podinfo-regex",
			"testdata/image/get_image_policy_regex.golden",
		},
	}

	for _, tc := range cases {
		cmd := cmdTestCase{
			args:   tc.args + " -n=" + namespace,
			assert: assertGoldenFile(tc.goldenFile),
		}
		cmd.runTestCmd(t)
	}
}
