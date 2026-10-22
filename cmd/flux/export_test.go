//go:build unit
// +build unit

/*
Copyright 2024 The Flux authors

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
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExport(t *testing.T) {
	namespace := allocateNamespace("flux-system")

	objectFile := "testdata/export/objects.yaml"
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
			"alert-provider",
			"export alert-provider slack",
			"testdata/export/provider.yaml",
			tmpl,
		},
		{
			"alert",
			"export alert flux-system",
			"testdata/export/alert.yaml",
			tmpl,
		},
		{
			"image policy",
			"export image policy flux-system",
			"testdata/export/image-policy.yaml",
			tmpl,
		},
		{
			"image repository",
			"export image repository flux-system",
			"testdata/export/image-repo.yaml",
			tmpl,
		},
		{
			"image update",
			"export image update flux-system",
			"testdata/export/image-update.yaml",
			tmpl,
		},
		{
			"source git",
			"export source git flux-system",
			"testdata/export/git-repo.yaml",
			tmpl,
		},
		{
			"source chart",
			"export source chart flux-system",
			"testdata/export/helm-chart.yaml",
			tmpl,
		},
		{
			"source helm",
			"export source helm flux-system",
			"testdata/export/helm-repo.yaml",
			tmpl,
		},
		{
			"receiver",
			"export receiver flux-system",
			"testdata/export/receiver.yaml",
			tmpl,
		},
		{
			"kustomization",
			"export kustomization flux-system",
			"testdata/export/ks.yaml",
			tmpl,
		},
		{
			"helmrelease",
			"export helmrelease flux-system",
			"testdata/export/helm-release.yaml",
			tmpl,
		},
		{
			"bucket",
			"export source bucket flux-system",
			"testdata/export/bucket.yaml",
			tmpl,
		},
		{
			"source external",
			"export source external flux-system",
			"testdata/export/external-artifact.yaml",
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

func TestExportAllWithCredentials(t *testing.T) {
	namespace := allocateNamespace("export-creds")
	tmpl := map[string]string{
		"fluxns": namespace,
	}
	testEnv.CreateObjectFile("testdata/export/with-credentials-objects.yaml", tmpl, t)

	cmd := cmdTestCase{
		args:   "export source git --all --with-credentials -n=" + namespace,
		assert: assertGoldenTemplateFile("testdata/export/with-credentials-all.yaml", tmpl),
	}
	cmd.runTestCmd(t)
}

func TestExportMultipleNames(t *testing.T) {
	namespace := allocateNamespace("export-multi")
	tmpl := map[string]string{
		"fluxns": namespace,
	}
	testEnv.CreateObjectFile("testdata/export/multi-objects.yaml", tmpl, t)
	t.Cleanup(func() { exportSourceWithCred = false })

	cases := []struct {
		name       string
		args       string
		goldenFile string
		wantErr    string
	}{
		{
			name:       "two names",
			args:       "export kustomization app-a app-b",
			goldenFile: "testdata/export/multi-ks.yaml",
		},
		{
			name:       "order and duplicates",
			args:       "export kustomization app-b app-a app-b",
			goldenFile: "testdata/export/multi-ks-reverse.yaml",
		},
		{
			name:       "one missing among several",
			args:       "export kustomization app-a missing app-b",
			goldenFile: "testdata/export/multi-ks-missing.yaml",
			wantErr:    "failed to export 1 of 3 Kustomization objects",
		},
		{
			name:       "all missing",
			args:       "export kustomization missing-a missing-b",
			goldenFile: "testdata/export/multi-ks-all-missing.yaml",
			wantErr:    "failed to export 2 of 2 Kustomization objects",
		},
		{
			name:    "single missing name unchanged",
			args:    "export kustomization missing",
			wantErr: `kustomizations.kustomize.toolkit.fluxcd.io "missing" not found`,
		},
		{
			name:    "duplicate missing name is a single name",
			args:    "export kustomization missing missing",
			wantErr: `kustomizations.kustomize.toolkit.fluxcd.io "missing" not found`,
		},
		{
			name:       "shared secret printed once",
			args:       "export source git repo-a repo-b --with-credentials",
			goldenFile: "testdata/export/multi-git-creds.yaml",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args + " -n=" + namespace,
				assert: assertGoldenTemplateFileAndError(tt.goldenFile, tmpl, tt.wantErr),
			}
			cmd.runTestCmd(t)
		})
	}
}

// assertGoldenTemplateFileAndError checks the output against the
// golden template file, or for empty output if goldenFile is empty,
// and the error against wantErr, or for success if wantErr is empty.
func assertGoldenTemplateFileAndError(goldenFile string, tmpl map[string]string, wantErr string) assertFunc {
	return func(output string, err error) error {
		if wantErr == "" {
			if err != nil {
				return fmt.Errorf("expected success but got error: %v", err)
			}
		} else if err == nil || err.Error() != wantErr {
			return fmt.Errorf("expected error %q but got %v", wantErr, err)
		}

		want := ""
		if goldenFile != "" {
			content, err := os.ReadFile(goldenFile)
			if err != nil {
				return fmt.Errorf("error reading golden file '%s': %w", goldenFile, err)
			}
			want, err = executeTemplate(string(content), tmpl)
			if err != nil {
				return err
			}
		}
		if diff := cmp.Diff(want, output); diff != "" {
			return fmt.Errorf("mismatch from golden file '%s' (-want +got):\n%s", goldenFile, diff)
		}
		return nil
	}
}
