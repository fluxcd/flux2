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

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetKubeconfigContextNamespace(t *testing.T) {
	tests := []struct {
		name           string
		kubeconfig     string
		context        string
		expectedResult string
	}{
		{
			name: "returns namespace from current context",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
    namespace: custom-ns
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			expectedResult: "custom-ns",
		},
		{
			name: "returns empty when context has no namespace",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			expectedResult: "",
		},
		{
			name: "returns namespace from context specified via --context flag",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: default-context
contexts:
- name: default-context
  context:
    cluster: my-cluster
    namespace: default-ns
- name: other-context
  context:
    cluster: my-cluster
    namespace: other-ns
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			context:        "other-context",
			expectedResult: "other-ns",
		},
		{
			name: "returns empty when context does not exist",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: non-existent
contexts: []
clusters: []
`,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Save and restore kubeconfigArgs state.
			origKubeConfig := kubeconfigArgs.KubeConfig
			origContext := kubeconfigArgs.Context
			t.Cleanup(func() {
				kubeconfigArgs.KubeConfig = origKubeConfig
				kubeconfigArgs.Context = origContext
			})

			// Write temporary kubeconfig.
			tmpDir := t.TempDir()
			kcPath := filepath.Join(tmpDir, "kubeconfig")
			g.Expect(os.WriteFile(kcPath, []byte(tt.kubeconfig), 0o600)).To(Succeed())
			kubeconfigArgs.KubeConfig = &kcPath

			if tt.context != "" {
				kubeconfigArgs.Context = &tt.context
			} else {
				empty := ""
				kubeconfigArgs.Context = &empty
			}

			got := getKubeconfigContextNamespace()
			g.Expect(got).To(Equal(tt.expectedResult))
		})
	}
}

func TestConfigureDefaultNamespace_ContextNamespace(t *testing.T) {
	tests := []struct {
		name              string
		kubeconfig        string
		envNamespace      string
		flagNamespace     string
		expectedNamespace string
	}{
		{
			name: "uses kubeconfig context namespace when no flag or env var is set",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
    namespace: context-ns
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			expectedNamespace: "context-ns",
		},
		{
			name: "env var takes precedence over kubeconfig context namespace",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
    namespace: context-ns
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			envNamespace:      "env-ns",
			expectedNamespace: "env-ns",
		},
		{
			name: "flag takes precedence over kubeconfig context namespace",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
    namespace: context-ns
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			flagNamespace:     "flag-ns",
			expectedNamespace: "flag-ns",
		},
		{
			name: "falls back to flux-system when context has no namespace",
			kubeconfig: `apiVersion: v1
kind: Config
current-context: my-context
contexts:
- name: my-context
  context:
    cluster: my-cluster
clusters:
- name: my-cluster
  cluster:
    server: https://localhost:6443
`,
			expectedNamespace: rootArgs.defaults.Namespace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Save and restore state.
			origKubeConfig := kubeconfigArgs.KubeConfig
			origContext := kubeconfigArgs.Context
			origNamespace := *kubeconfigArgs.Namespace
			t.Cleanup(func() {
				kubeconfigArgs.KubeConfig = origKubeConfig
				kubeconfigArgs.Context = origContext
				*kubeconfigArgs.Namespace = origNamespace
				os.Unsetenv("FLUX_SYSTEM_NAMESPACE")
			})

			// Write temporary kubeconfig.
			tmpDir := t.TempDir()
			kcPath := filepath.Join(tmpDir, "kubeconfig")
			g.Expect(os.WriteFile(kcPath, []byte(tt.kubeconfig), 0o600)).To(Succeed())
			kubeconfigArgs.KubeConfig = &kcPath

			emptyCtx := ""
			kubeconfigArgs.Context = &emptyCtx

			if tt.envNamespace != "" {
				t.Setenv("FLUX_SYSTEM_NAMESPACE", tt.envNamespace)
			}

			// Reset default namespace and re-run configuration.
			configureDefaultNamespace()

			// Simulate PersistentPreRunE behavior.
			if tt.flagNamespace != "" {
				// When a flag is explicitly provided, PersistentPreRunE
				// sees Changed("namespace") == true, so context is skipped.
				// Simulate by directly setting the value.
				*kubeconfigArgs.Namespace = tt.flagNamespace
			} else {
				// Simulate the PersistentPreRunE logic when no --namespace flag is set.
				if os.Getenv("FLUX_SYSTEM_NAMESPACE") == "" {
					if ctxNs := getKubeconfigContextNamespace(); ctxNs != "" {
						*kubeconfigArgs.Namespace = ctxNs
					}
				}
			}

			g.Expect(*kubeconfigArgs.Namespace).To(Equal(tt.expectedNamespace))
		})
	}
}
