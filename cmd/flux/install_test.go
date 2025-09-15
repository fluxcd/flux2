/*
Copyright 2025 The Flux authors

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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	ssautil "github.com/fluxcd/pkg/ssa/utils"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
)

func TestInstall(t *testing.T) {
	// The pointer to kubeconfigArgs.Namespace is shared across
	// the tests. When a new value is set, it will linger and
	// impact subsequent tests.
	// Given that this test uses an invalid namespace, it ensures
	// to restore whatever value it had previously.
	currentNamespace := *kubeconfigArgs.Namespace
	t.Cleanup(func() { *kubeconfigArgs.Namespace = currentNamespace })

	tests := []struct {
		name   string
		args   string
		assert assertFunc
	}{
		{
			name:   "invalid namespace",
			args:   "install --namespace='@#[]'",
			assert: assertError("namespace must be a valid DNS label: \"@#[]\""),
		},
		{
			name:   "invalid sub-command",
			args:   "install unexpectedPosArg --namespace=example",
			assert: assertError(`unknown command "unexpectedPosArg" for "flux install"`),
		},
		{
			name:   "missing image pull secret",
			args:   "install --registry-creds=fluxcd:test",
			assert: assertError(`--registry-creds requires --image-pull-secret to be set`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdTestCase{
				args:   tt.args,
				assert: tt.assert,
			}
			cmd.runTestCmd(t)
		})
	}
}

func TestInstall_ComponentsExtra(t *testing.T) {
	g := NewWithT(t)
	command := "install --export --components-extra=" +
		strings.Join(install.MakeDefaultOptions().ComponentsExtra, ",")

	output, err := executeCommand(command)
	g.Expect(err).NotTo(HaveOccurred())

	manifests, err := ssautil.ReadObjects(strings.NewReader(output))
	g.Expect(err).NotTo(HaveOccurred())

	foundImageAutomation := false
	foundImageReflector := false
	foundSourceWatcher := false
	foundExternalArtifact := false
	for _, obj := range manifests {
		if obj.GetKind() == "Deployment" && obj.GetName() == "image-automation-controller" {
			foundImageAutomation = true
		}
		if obj.GetKind() == "Deployment" && obj.GetName() == "image-reflector-controller" {
			foundImageReflector = true
		}
		if obj.GetKind() == "Deployment" && obj.GetName() == "source-watcher" {
			foundSourceWatcher = true
		}
		if obj.GetKind() == "Deployment" &&
			(obj.GetName() == "kustomize-controller" || obj.GetName() == "helm-controller") {
			containers, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
			g.Expect(containers).ToNot(BeEmpty())
			args, _, _ := unstructured.NestedSlice(containers[0].(map[string]any), "args")
			g.Expect(args).To(ContainElement("--feature-gates=ExternalArtifact=true"))
			foundExternalArtifact = true
		}
	}
	g.Expect(foundImageAutomation).To(BeTrue(), "image-automation-controller deployment not found")
	g.Expect(foundImageReflector).To(BeTrue(), "image-reflector-controller deployment not found")
	g.Expect(foundSourceWatcher).To(BeTrue(), "source-watcher deployment not found")
	g.Expect(foundExternalArtifact).To(BeTrue(), "ExternalArtifact feature gate not found")
}
