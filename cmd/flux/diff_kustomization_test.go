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
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fluxcd/flux2/internal/kustomization"
	"github.com/fluxcd/pkg/ssa"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDiffKustomization(t *testing.T) {
	tests := []struct {
		name       string
		args       string
		objectFile string
		assert     assertFunc
	}{
		{
			name:       "no args",
			args:       "diff kustomization podinfo",
			objectFile: "",
			assert:     assertError("invalid resource path \"\""),
		},
		{
			name:       "diff nothing deployed",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "",
			assert:     assertGoldenFile("./testdata/diff-kustomization/nothing-is-deployed.golden"),
		},
		{
			name:       "diff with a deployment object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "./testdata/diff-kustomization/deployment.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-deployment.golden"),
		},
		{
			name:       "diff with a drifted service object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "./testdata/diff-kustomization/service.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-service.golden"),
		},
		{
			name:       "diff with a drifted secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "./testdata/diff-kustomization/secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-secret.golden"),
		},
		{
			name:       "diff with a drifted key in sops secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "./testdata/diff-kustomization/key-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-key-sops-secret.golden"),
		},
		{
			name:       "diff with a drifted value in sops secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo",
			objectFile: "./testdata/diff-kustomization/value-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-value-sops-secret.golden"),
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}

	b, _ := kustomization.NewBuilder(kubeconfigArgs, "podinfo", "")

	resourceManager, err := b.Manager()
	if err != nil {
		t.Fatal(err)
	}

	setup(t, tmpl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.objectFile != "" {
				resourceManager.ApplyAll(context.Background(), createObjectFromFile(tt.objectFile, tmpl, t), ssa.DefaultApplyOptions())
			}
			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
			if tt.objectFile != "" {
				testEnv.DeleteObjectFile(tt.objectFile, tmpl, t)
			}
		})
	}
}

func createObjectFromFile(objectFile string, templateValues map[string]string, t *testing.T) []*unstructured.Unstructured {
	buf, err := os.ReadFile(objectFile)
	if err != nil {
		t.Fatalf("Error reading file '%s': %v", objectFile, err)
	}
	content, err := executeTemplate(string(buf), templateValues)
	if err != nil {
		t.Fatalf("Error evaluating template file '%s': '%v'", objectFile, err)
	}
	clientObjects, err := readYamlObjects(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Error decoding yaml file '%s': %v", objectFile, err)
	}

	return clientObjects
}
