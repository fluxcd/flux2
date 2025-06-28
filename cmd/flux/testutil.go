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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattn/go-shellwords"
)

// setupTestEnvironment sets up the test environment for flux tests
func setupTestEnvironment(t *testing.T) {
	// Create necessary directories and files for tests
	if err := ensureManifestDirectories(); err != nil {
		t.Fatalf("Failed to set up test environment: %v", err)
	}
}

// executeCommandWithIn executes a command with the provided input
func executeCommandWithIn(cmd string, in io.Reader) (string, error) {
	defer resetCmdArgs()
	args, err := shellwords.Parse(cmd)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)

	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	if in != nil {
		rootCmd.SetIn(in)
	}

	_, err = rootCmd.ExecuteC()
	result := buf.String()

	return result, err
}

// createTempKubeconfig creates a temporary kubeconfig file for testing
func createTempKubeconfig(t *testing.T, content string) string {
	tmpDir, err := os.MkdirTemp("", "flux-kubeconfig-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	
	kubeconfigPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}
	
	return kubeconfigPath
}

// savePreviousEnv saves the current environment variable and returns a function to restore it
func savePreviousEnv(t *testing.T, key string) func() {
	previous := os.Getenv(key)
	return func() {
		os.Setenv(key, previous)
	}
}
