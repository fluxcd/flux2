// +build e2e

package main

import "testing"

func TestImageScanning(t *testing.T) {
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

	namespace := "tis"
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
