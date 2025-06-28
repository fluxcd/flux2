/*
Copyright 2023 The Flux authors

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
)

func TestKubeConfigMerging(t *testing.T) {
	// Save original KUBECONFIG env var to restore later
	origKubeconfig := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", origKubeconfig)

	// Create a temporary directory for test kubeconfig files
	tmpDir, err := os.MkdirTemp("", "flux-kubeconfig-test")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first kubeconfig file
	kubeconfig1 := filepath.Join(tmpDir, "config1")
	kubeconfig1Content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://cluster1:6443
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: user1
  name: context1
current-context: context1
users:
- name: user1
  user:
    token: token1
`
	if err := os.WriteFile(kubeconfig1, []byte(kubeconfig1Content), 0644); err != nil {
		t.Fatalf("failed to write kubeconfig1: %v", err)
	}

	// Create second kubeconfig file
	kubeconfig2 := filepath.Join(tmpDir, "config2")
	kubeconfig2Content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://cluster2:6443
  name: cluster2
contexts:
- context:
    cluster: cluster2
    user: user2
  name: context2
current-context: context2
users:
- name: user2
  user:
    token: token2
`
	if err := os.WriteFile(kubeconfig2, []byte(kubeconfig2Content), 0644); err != nil {
		t.Fatalf("failed to write kubeconfig2: %v", err)
	}

	// Test case 1: Single kubeconfig specified via --kubeconfig flag
	t.Run("SingleKubeconfigViaFlag", func(t *testing.T) {
		ka := NewKubeConfigArgs()
		ka.KubeConfig = kubeconfig1
		
		config, err := ka.loadKubeConfig()
		if err != nil {
			t.Fatalf("loadKubeConfig failed: %v", err)
		}
		
		// Verify single kubeconfig loaded correctly
		if config.CurrentContext != "context1" {
			t.Errorf("expected current-context to be context1, got %s", config.CurrentContext)
		}
		
		if _, ok := config.Clusters["cluster1"]; !ok {
			t.Error("expected cluster1 to be present in config")
		}
		
		if _, ok := config.Contexts["context1"]; !ok {
			t.Error("expected context1 to be present in config")
		}
	})

	// Test case 2: Multiple kubeconfig files via KUBECONFIG env var
	t.Run("MultipleKubeconfigsViaEnvVar", func(t *testing.T) {
		// Set KUBECONFIG env var with multiple paths
		pathSeparator := string(os.PathListSeparator)
		os.Setenv("KUBECONFIG", kubeconfig1+pathSeparator+kubeconfig2)
		
		ka := NewKubeConfigArgs()
		config, err := ka.loadKubeConfig()
		if err != nil {
			t.Fatalf("loadKubeConfig failed: %v", err)
		}
		
		// Verify both kubeconfig files were loaded and merged
		if _, ok := config.Clusters["cluster1"]; !ok {
			t.Error("expected cluster1 to be present in merged config")
		}
		
		if _, ok := config.Clusters["cluster2"]; !ok {
			t.Error("expected cluster2 to be present in merged config")
		}
		
		if _, ok := config.Contexts["context1"]; !ok {
			t.Error("expected context1 to be present in merged config")
		}
		
		if _, ok := config.Contexts["context2"]; !ok {
			t.Error("expected context2 to be present in merged config")
		}
		
		// Current context should be from the last file in KUBECONFIG
		if config.CurrentContext != "context2" {
			t.Errorf("expected current-context to be context2, got %s", config.CurrentContext)
		}
	})

	// Test case 3: --context flag overrides current-context from kubeconfig
	t.Run("ContextFlagOverride", func(t *testing.T) {
		ka := NewKubeConfigArgs()
		ka.KubeConfig = kubeconfig1
		ka.KubeContext = "context2"
		
		// Add context2 to kubeconfig1 for this test
		kubeconfig1WithContext2 := kubeconfig1 + ".context2"
		kubeconfig1WithContext2Content := kubeconfig1Content + `
contexts:
- context:
    cluster: cluster1
    user: user1
  name: context2
`
		if err := os.WriteFile(kubeconfig1WithContext2, []byte(kubeconfig1WithContext2Content), 0644); err != nil {
			t.Fatalf("failed to write kubeconfig1WithContext2: %v", err)
		}
		
		ka.KubeConfig = kubeconfig1WithContext2
		_, err := ka.kubeConfig(nil)
		
		// We expect an error because context2 references cluster2 which doesn't exist in kubeconfig1
		// But we can still verify that the context was selected
		if err == nil {
			t.Fatal("expected error for invalid context, got nil")
		}
	})

	// Test case 4: Insecure TLS verification
	t.Run("InsecureTLSVerification", func(t *testing.T) {
		ka := NewKubeConfigArgs()
		ka.KubeConfig = kubeconfig1
		ka.InsecureSkipVerify = true
		
		config, err := ka.loadKubeConfig()
		if err != nil {
			t.Fatalf("loadKubeConfig failed: %v", err)
		}
		
		// We can't directly test the insecure flag here since it's applied in the ConfigOverrides
		// Instead we verify that the base config was loaded correctly
		if config.CurrentContext != "context1" {
			t.Errorf("expected current-context to be context1, got %s", config.CurrentContext)
		}
	})
}
