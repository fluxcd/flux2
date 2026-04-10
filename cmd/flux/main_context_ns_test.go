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

func TestContextNamespaceOptIn(t *testing.T) {
	kubeconfig := `apiVersion: v1
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
`

	tests := []struct {
		name              string
		nsFollowsFlag    bool
		nsFollowsEnv     string
		envNamespace      string
		flagNamespace     string
		expectedNamespace string
	}{
		{
			name:              "ignores context namespace when not opted in",
			expectedNamespace: rootArgs.defaults.Namespace,
		},
		{
			name:              "uses context namespace when opted in via flag",
			nsFollowsFlag:    true,
			expectedNamespace: "context-ns",
		},
		{
			name:              "uses context namespace when opted in via env var",
			nsFollowsEnv:     "1",
			expectedNamespace: "context-ns",
		},
		{
			name:              "FLUX_SYSTEM_NAMESPACE takes precedence over context namespace",
			nsFollowsFlag:    true,
			envNamespace:      "env-ns",
			expectedNamespace: "env-ns",
		},
		{
			name:              "--namespace flag takes precedence over context namespace",
			nsFollowsFlag:    true,
			flagNamespace:     "flag-ns",
			expectedNamespace: "flag-ns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Save and restore state.
			origKubeConfig := kubeconfigArgs.KubeConfig
			origContext := kubeconfigArgs.Context
			origNamespace := *kubeconfigArgs.Namespace
			origNsFollows := rootArgs.nsFollowsKubeContext
			t.Cleanup(func() {
				kubeconfigArgs.KubeConfig = origKubeConfig
				kubeconfigArgs.Context = origContext
				*kubeconfigArgs.Namespace = origNamespace
				rootArgs.nsFollowsKubeContext = origNsFollows
				os.Unsetenv("FLUX_SYSTEM_NAMESPACE")
				os.Unsetenv("FLUX_NS_FOLLOWS_KUBE_CONTEXT")
			})

			// Write temporary kubeconfig.
			tmpDir := t.TempDir()
			kcPath := filepath.Join(tmpDir, "kubeconfig")
			g.Expect(os.WriteFile(kcPath, []byte(kubeconfig), 0o600)).To(Succeed())
			kubeconfigArgs.KubeConfig = &kcPath

			emptyCtx := ""
			kubeconfigArgs.Context = &emptyCtx

			rootArgs.nsFollowsKubeContext = tt.nsFollowsFlag
			if tt.nsFollowsEnv != "" {
				t.Setenv("FLUX_NS_FOLLOWS_KUBE_CONTEXT", tt.nsFollowsEnv)
			}
			if tt.envNamespace != "" {
				t.Setenv("FLUX_SYSTEM_NAMESPACE", tt.envNamespace)
			}

			// Reset default namespace and re-run configuration.
			configureDefaultNamespace()

			// Simulate PersistentPreRunE behavior.
			if tt.flagNamespace != "" {
				*kubeconfigArgs.Namespace = tt.flagNamespace
			} else if (rootArgs.nsFollowsKubeContext || os.Getenv("FLUX_NS_FOLLOWS_KUBE_CONTEXT") != "") &&
				os.Getenv("FLUX_SYSTEM_NAMESPACE") == "" {
				if ctxNs := getKubeconfigContextNamespace(); ctxNs != "" {
					*kubeconfigArgs.Namespace = ctxNs
				}
			}

			g.Expect(*kubeconfigArgs.Namespace).To(Equal(tt.expectedNamespace))
		})
	}
}
