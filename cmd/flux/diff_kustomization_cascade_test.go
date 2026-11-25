//go:build unit
// +build unit

/*
Copyright 2026 The Flux authors

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

func TestDiffKustomizationRecursiveCascadeDelete(t *testing.T) {
	fluxNS := allocateNamespace("flux-system")
	setupTestNamespace(fluxNS, t)

	tmpl := map[string]string{
		"fluxns": fluxNS,
	}
	testEnv.CreateObjectFile("./testdata/diff-kustomization/cascade-delete.yaml", tmpl, t)

	cmd := cmdTestCase{
		args: "diff kustomization cascade-root " +
			"--path ./testdata/build-kustomization/cascade-empty " +
			"--recursive --progress-bar=false -n " + fluxNS,
		assert: assertGoldenTemplateFile(
			"./testdata/diff-kustomization/diff-with-recursive-cascade-delete.golden",
			tmpl,
		),
	}
	cmd.runTestCmd(t)
}
