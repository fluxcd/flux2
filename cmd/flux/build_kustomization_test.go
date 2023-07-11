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
			args:       "build kustomization podinfo --kustomization-file ./testdata/build-kustomization/podinfo.yaml --path ./testdata/build-kustomization/podinfo",
			resultFile: "./testdata/build-kustomization/podinfo-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build podinfo without service",
			args:       "build kustomization podinfo --kustomization-file ./testdata/build-kustomization/podinfo.yaml --path ./testdata/build-kustomization/delete-service",
			resultFile: "./testdata/build-kustomization/podinfo-without-service-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build deployment and configmap with var substitution",
			args:       "build kustomization podinfo --kustomization-file ./testdata/build-kustomization/podinfo.yaml --path ./testdata/build-kustomization/var-substitution",
			resultFile: "./testdata/build-kustomization/podinfo-with-var-substitution-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
		{
			name:       "build deployment and configmap with var substitution in dry-run mode",
			args:       "build kustomization podinfo --kustomization-file ./testdata/build-kustomization/podinfo.yaml --path ./testdata/build-kustomization/var-substitution --dry-run",
			resultFile: "./testdata/build-kustomization/podinfo-with-var-substitution-result.yaml",
			assertFunc: "assertGoldenTemplateFile",
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setup(t, tmpl)

	testEnv.CreateObjectFile("./testdata/build-kustomization/podinfo-source.yaml", tmpl, t)

	temp, err := template.New("podinfo").Parse(podinfo)
	if err != nil {
		t.Fatal(err)
	}

	var b bytes.Buffer
	err = temp.Execute(&b, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile("./testdata/build-kustomization/podinfo.yaml", b.Bytes(), 0666)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove("./testdata/build-kustomization/podinfo.yaml") })

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
