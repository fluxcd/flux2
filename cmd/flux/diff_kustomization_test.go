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

	"github.com/fluxcd/flux2/v2/internal/build"
	"github.com/fluxcd/pkg/ssa"
	"github.com/fluxcd/pkg/ssa/normalize"
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
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-new-kustomization.golden"),
		},
		{
			name:       "diff with a deployment object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/deployment.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-deployment.golden"),
		},
		{
			name:       "diff with a drifted service object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/service.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-service.golden"),
		},
		{
			name:       "diff with a drifted secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-secret.golden"),
		},
		{
			name:       "diff with a drifted key in sops secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/key-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-key-sops-secret.golden"),
		},
		{
			name:       "diff with a drifted value in sops secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/value-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-value-sops-secret.golden"),
		},
		{
			name:       "diff with a sops dockerconfigjson secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/dockerconfigjson-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-dockerconfigjson-sops-secret.golden"),
		},
		{
			name:       "diff with a sops stringdata secret object",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false",
			objectFile: "./testdata/diff-kustomization/stringdata-sops-secret.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-drifted-stringdata-sops-secret.golden"),
		},
		{
			name:       "diff where kustomization file has multiple objects with the same name",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false --kustomization-file ./testdata/diff-kustomization/flux-kustomization-multiobj.yaml",
			objectFile: "",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-new-kustomization.golden"),
		},
		{
			name:       "diff with recursive",
			args:       "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo-with-my-app --progress-bar=false --recursive --local-sources GitRepository/default/podinfo=./testdata/build-kustomization",
			objectFile: "./testdata/diff-kustomization/my-app.yaml",
			assert:     assertGoldenFile("./testdata/diff-kustomization/diff-with-recursive.golden"),
		},
	}

	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}

	b, _ := build.NewBuilder("podinfo", "", build.WithClientConfig(kubeconfigArgs, kubeclientOptions))

	resourceManager, err := b.Manager()
	if err != nil {
		t.Fatal(err)
	}

	setup(t, tmpl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.objectFile != "" {
				if _, err := resourceManager.ApplyAll(context.Background(), createObjectFromFile(tt.objectFile, tmpl, t), ssa.DefaultApplyOptions()); err != nil {
					t.Error(err)
				}
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

// TestDiffKustomizationNotDeployed tests `flux diff ks` when the Kustomization
// CR does not exist in the cluster but is provided via --kustomization-file.
// Reproduces https://github.com/fluxcd/flux2/issues/5439
func TestDiffKustomizationNotDeployed(t *testing.T) {
	// Use a dedicated namespace with NO setup() -- the Kustomization CR
	// intentionally does not exist in the cluster.
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupTestNamespace(tmpl["fluxns"], t)

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name: "fails without --ignore-not-found",
			args: "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false " +
				"--kustomization-file ./testdata/diff-kustomization/flux-kustomization-local-only.yaml",
			assert: assertError("failed to get kustomization object: kustomizations.kustomize.toolkit.fluxcd.io \"podinfo\" not found"),
		},
		{
			name: "succeeds with --ignore-not-found and --kustomization-file",
			args: "diff kustomization podinfo --path ./testdata/build-kustomization/podinfo --progress-bar=false " +
				"--kustomization-file ./testdata/diff-kustomization/flux-kustomization-local-only.yaml " +
				"--ignore-not-found",
			assert: assertGoldenFile("./testdata/diff-kustomization/diff-new-kustomization.golden"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args + " -n " + tmpl["fluxns"],
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}

// TestDiffKustomizationTakeOwnership tests `flux diff ks` when taking ownership
// of existing resources on the cluster. A "pre-existing" configmap is applied
// to the cluster, and the kustomization contains a matching configmap; the
// diff should show the labels added by flux
func TestDiffKustomizationTakeOwnership(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupTestNamespace(tmpl["fluxns"], t)

	b, _ := build.NewBuilder("configmaps", "", build.WithClientConfig(kubeconfigArgs, kubeclientOptions))
	resourceManager, err := b.Manager()
	if err != nil {
		t.Fatal(err)
	}

	// Pre-create the "existing" configmap in the cluster without Flux labels
	if _, err := resourceManager.ApplyAll(context.Background(), createObjectFromFile("./testdata/diff-kustomization/existing-configmap.yaml", tmpl, t), ssa.DefaultApplyOptions()); err != nil {
		t.Fatal(err)
	}

	cmd := cmdTestCase{
		args: "diff kustomization configmaps --path ./testdata/build-kustomization/configmaps --progress-bar=false " +
			"--kustomization-file ./testdata/diff-kustomization/flux-kustomization-configmaps.yaml " +
			"--ignore-not-found" +
			" -n " + tmpl["fluxns"],
		assert: assertGoldenFile("./testdata/diff-kustomization/diff-taking-ownership.golden"),
	}
	cmd.runTestCmd(t)
}

// TestDiffKustomizationNewNamespaceAndConfigmap runs `flux diff ks` when the
// kustomization creates a new namespace and resources inside it. The server-side
// dry-run cannot resolve resources in a namespace that doesn't exist yet,
// consistent with `kubectl diff --server-side` behavior.
func TestDiffKustomizationNewNamespaceAndConfigmap(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupTestNamespace(tmpl["fluxns"], t)

	cmd := cmdTestCase{
		args: "diff kustomization new-namespace-and-configmap --path ./testdata/build-kustomization/new-namespace-and-configmap --progress-bar=false " +
			"--kustomization-file ./testdata/diff-kustomization/flux-kustomization-new-namespace-and-configmap.yaml " +
			"--ignore-not-found" +
			" -n " + tmpl["fluxns"],
		assert: assertError("ConfigMap/new-ns/app-config not found: namespaces \"new-ns\" not found"),
	}
	cmd.runTestCmd(t)
}

// TestDiffKustomizationNewNamespaceOnly runs `flux diff ks` when the
// kustomization creates only a new namespace. The diff should show the
// namespace as created.
func TestDiffKustomizationNewNamespaceOnly(t *testing.T) {
	tmpl := map[string]string{
		"fluxns": allocateNamespace("flux-system"),
	}
	setupTestNamespace(tmpl["fluxns"], t)

	cmd := cmdTestCase{
		args: "diff kustomization new-namespace-only --path ./testdata/build-kustomization/new-namespace-only --progress-bar=false " +
			"--kustomization-file ./testdata/diff-kustomization/flux-kustomization-new-namespace-only.yaml " +
			"--ignore-not-found" +
			" -n " + tmpl["fluxns"],
		assert: assertGoldenFile("./testdata/diff-kustomization/diff-new-namespace-only.golden"),
	}
	cmd.runTestCmd(t)
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

	if err := normalize.UnstructuredList(clientObjects); err != nil {
		t.Fatalf("Error setting native kinds defaults for '%s': %v", objectFile, err)
	}

	return clientObjects
}
