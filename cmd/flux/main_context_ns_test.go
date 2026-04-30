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
	"k8s.io/cli-runtime/pkg/genericclioptions"
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

			// Write temporary kubeconfig.
			tmpDir := t.TempDir()
			kcPath := filepath.Join(tmpDir, "kubeconfig")
			g.Expect(os.WriteFile(kcPath, []byte(tt.kubeconfig), 0o600)).To(Succeed())

			// Use a local ConfigFlags instance to avoid polluting the
			// package-global kubeconfigArgs (which caches a clientConfig
			// internally and would leak state across tests).
			cf := genericclioptions.NewConfigFlags(false)
			cf.KubeConfig = &kcPath
			cf.Context = &tt.context

			got := getKubeconfigContextNamespace(cf)
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
		nsFollowsFlag     bool
		nsFollowsEnv      string
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
			nsFollowsFlag:     true,
			expectedNamespace: "context-ns",
		},
		{
			name:              "uses context namespace when opted in via env var",
			nsFollowsEnv:      "1",
			expectedNamespace: "context-ns",
		},
		{
			name:              "context namespace takes precedence over FLUX_SYSTEM_NAMESPACE when opted in",
			nsFollowsFlag:     true,
			envNamespace:      "env-ns",
			expectedNamespace: "context-ns",
		},
		{
			name:              "FLUX_SYSTEM_NAMESPACE used when not opted in",
			envNamespace:      "env-ns",
			expectedNamespace: "env-ns",
		},
		{
			name:              "--namespace flag takes precedence over context namespace",
			nsFollowsFlag:     true,
			flagNamespace:     "flag-ns",
			expectedNamespace: "flag-ns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Write temporary kubeconfig.
			tmpDir := t.TempDir()
			kcPath := filepath.Join(tmpDir, "kubeconfig")
			g.Expect(os.WriteFile(kcPath, []byte(kubeconfig), 0o600)).To(Succeed())

			// Use a local ConfigFlags instance to avoid polluting the
			// package-global kubeconfigArgs.
			cf := genericclioptions.NewConfigFlags(false)
			cf.KubeConfig = &kcPath
			emptyCtx := ""
			cf.Context = &emptyCtx

			// Mirror configureDefaultNamespace behavior on the local instance.
			defaultNs := rootArgs.defaults.Namespace
			cf.Namespace = &defaultNs

			if tt.envNamespace != "" {
				t.Setenv("FLUX_SYSTEM_NAMESPACE", tt.envNamespace)
				envNs := tt.envNamespace
				cf.Namespace = &envNs
			}
			if tt.nsFollowsEnv != "" {
				t.Setenv("FLUX_NS_FOLLOWS_KUBE_CONTEXT", tt.nsFollowsEnv)
			}

			// Simulate PersistentPreRunE behavior.
			if tt.flagNamespace != "" {
				*cf.Namespace = tt.flagNamespace
			} else if tt.nsFollowsFlag || os.Getenv("FLUX_NS_FOLLOWS_KUBE_CONTEXT") != "" {
				if ctxNs := getKubeconfigContextNamespace(cf); ctxNs != "" {
					*cf.Namespace = ctxNs
				}
			}

			g.Expect(*cf.Namespace).To(Equal(tt.expectedNamespace))
		})
	}
}
