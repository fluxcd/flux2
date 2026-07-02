//go:build unit
// +build unit

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

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"
)

func setup(t *testing.T, tmpl map[string]string) {
	t.Helper()
	testEnv.CreateObjectFile("./testdata/build-kustomization/podinfo-source.yaml", tmpl, t)
	testEnv.CreateObjectFile("./testdata/build-kustomization/podinfo-kustomization.yaml", tmpl, t)
}

func TestBuildKustomization(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		resultFile string
		assertFunc string
	}{
		{
			name:       "no args",
			args:       "build kustomization podinfo",
			resultFile: "invalid resource path \"\"",
			assertFunc: "assertError",
		},
		{
			name:       "build podinfo",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build podinfo (in-memory)",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/podinfo --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build podinfo without service",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/delete-service",
			resultFile: "./testdata/build-kustomization/podinfo-without-service-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build deployment and configmap with var substitution",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/var-substitution",
			resultFile: "./testdata/build-kustomization/podinfo-with-var-substitution-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build ignore",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/ignore --ignore-paths \"!configmap.yaml,!secret.yaml\"",
			resultFile: "./testdata/build-kustomization/podinfo-with-ignore-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build ignore (in-memory)",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/ignore --ignore-paths \"!configmap.yaml,!secret.yaml\" --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-with-ignore-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive (in-memory)",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setup(t, tmpl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var assert assertFunc

			switch tt.assertFunc {
			case "assertGoldenTemplateFile":
				assert = assertGoldenTemplateFile(tt.resultFile, tmpl)
			case "assertError":
				assert = assertError(tt.resultFile)
			}

			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: assert,
			}

			cmd.runTestCmd(t)
		})
	}
}

func TestBuildLocalKustomization(t *testing.T) {
	podinfo := `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: podinfo
  namespace: {{ .fluxns }}
spec:
  interval: 5m0s
  path: ./kustomize
  force: true
  prune: true
  sourceRef:
    kind: GitRepository
    name: podinfo
  targetNamespace: default
  postBuild:
    substitute:
      cluster_env: "prod"
      cluster_region: "eu-central-1"
`

	tmpFile := filepath.Join(t.TempDir(), "podinfo.yaml")

	tests := []struct {
		name       string
		args       string
		resultFile string
		assertFunc string
	}{
		{
			name:       "no args",
			args:       "build kustomization podinfo --kustomization-file ./wrongfile/ --path ./testdata/build-kustomization/podinfo",
			resultFile: "invalid kustomization file \"./wrongfile/\"",
			assertFunc: "assertError",
		},
		{
			name:       "build podinfo",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build podinfo (in-memory)",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build podinfo without service",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/delete-service",
			resultFile: "./testdata/build-kustomization/podinfo-without-service-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build deployment and configmap with var substitution",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/var-substitution",
			resultFile: "./testdata/build-kustomization/podinfo-with-var-substitution-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build deployment and configmap with var substitution in dry-run mode",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/var-substitution --dry-run",
			resultFile: "./testdata/build-kustomization/podinfo-with-var-substitution-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive in dry-run mode",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization --dry-run",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive (in-memory)",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with recursive in dry-run mode (in-memory)",
			args:       "build kustomization podinfo --kustomization-file " + tmpFile + " --path ./testdata/build-kustomization/podinfo-with-my-app --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization --in-memory-build=true --dry-run",
			resultFile: "./testdata/build-kustomization/podinfo-with-my-app-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setup(t, tmpl)

	temp, err := template.New("podinfo").Parse(podinfo)
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	err = temp.Execute(&b, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(tmpFile, b.Bytes(), 0666)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var assert assertFunc

			switch tt.assertFunc {
			case "assertGoldenTemplateFile":
				assert = assertGoldenTemplateFile(tt.resultFile, tmpl)
			case "assertError":
				assert = assertError(tt.resultFile)
			}

			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: assert,
			}

			cmd.runTestCmd(t)
		})
	}
}

func TestBuildKustomizationDefaultBuildsAbsolutePathOutsideCwd(t *testing.T) {
	cwdDir, sourceDir, kustomizationFile := newOutsideCwdKustomization(t, "default", "")

	restore := chdirForTest(t, cwdDir)
	defer restore()

	flag := buildKsCmd.Flags().Lookup("in-memory-build")
	if flag == nil {
		t.Fatal("missing in-memory-build flag")
	}
	defaultValue, err := strconv.ParseBool(flag.DefValue)
	if err != nil {
		t.Fatal(err)
	}
	buildKsArgs.inMemoryBuild = defaultValue

	output, err := executeCommand("build kustomization app --path " + sourceDir +
		" --kustomization-file " + kustomizationFile +
		" --namespace default --dry-run")
	if err != nil {
		t.Fatalf("expected build to succeed with default backend, got: %v", err)
	}
	if !strings.Contains(output, "name: outside-cwd") {
		t.Fatalf("expected rendered ConfigMap in output, got:\n%s", output)
	}
}

func chdirForTest(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	}
}

func newOutsideCwdKustomization(t *testing.T, namespace, targetNamespace string) (string, string, string) {
	t.Helper()
	parentDir := t.TempDir()

	cwdDir := filepath.Join(parentDir, "cwd")
	if err := os.MkdirAll(cwdDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sourceDir := filepath.Join(parentDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "kustomization.yaml"), []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(sourceDir, "configmap.yaml"), []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: outside-cwd
`), 0o644); err != nil {
		t.Fatal(err)
	}

	targetNamespaceField := ""
	if targetNamespace != "" {
		targetNamespaceField = "\n  targetNamespace: " + targetNamespace
	}

	kustomizationFile := filepath.Join(parentDir, "app.yaml")
	if err := os.WriteFile(kustomizationFile, []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: app
  namespace: `+namespace+`
spec:
  interval: 5m
  path: ./source
  prune: true`+targetNamespaceField+`
  sourceRef:
    kind: GitRepository
    name: app
`), 0o644); err != nil {
		t.Fatal(err)
	}

	return cwdDir, sourceDir, kustomizationFile
}

// TestBuildKustomizationPathNormalization verifies that absolute and complex
// paths are normalized to prevent path concatenation bugs (issue #5673).
// Without normalization, paths could be duplicated like: /path/test/path/test/file
func TestBuildKustomizationPathNormalization(t *testing.T) {
	// Get absolute path to testdata to test absolute path handling
	absTestDataPath, err := filepath.Abs("testdata/build-kustomization/podinfo")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	tests := []struct {
		name       string
		args       string
		resultFile string
		assertFunc string
	}{
		{
			name:       "build with absolute path",
			args:       "build kustomization podinfo --path " + absTestDataPath,
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with absolute path (in-memory)",
			args:       "build kustomization podinfo --path " + absTestDataPath + " --in-memory-build=true",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with complex relative path (parent dir)",
			args:       "build kustomization podinfo --path ./testdata/build-kustomization/../build-kustomization/podinfo",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build with path containing redundant separators",
			args:       "build kustomization podinfo --path ./testdata//build-kustomization//podinfo",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setup(t, tmpl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var assert assertFunc

			switch tt.assertFunc {
			case "assertGoldenTemplateFile":
				assert = assertGoldenTemplateFile(tt.resultFile, tmpl)
			case "assertError":
				assert = assertError(tt.resultFile)
			}

			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: assert,
			}

			cmd.runTestCmd(t)
		})
	}
}
