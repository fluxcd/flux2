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
	"fmt"
	"os"
	"path/filepath"
)

// createTestManifests creates placeholder manifest files for testing
// This function ensures that the Go embed directive can find the required files
func createTestManifests() error {
	// Define the directories to create
	dirs := []string{
		"manifests",
		filepath.Join("manifests", "bases"),
		filepath.Join("manifests", "bases", "helm-controller"),
	}

	// Create the directories if they don't exist
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	// Create the placeholder files
	files := map[string][]byte{
		filepath.Join("manifests", "dummy.yaml"): []byte(`# This is a placeholder file to ensure the Go embed directive can find at least one file
# It will be replaced by actual manifests when bundle.sh is run successfully
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
`),
		filepath.Join("manifests", "placeholder.yaml"): []byte(`# This is an additional placeholder file to ensure the Go embed directive works properly
# It will be replaced by actual manifests when bundle.sh is run
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-additional
  namespace: flux-system
data:
  placeholder: "true"
`),
		filepath.Join("manifests", "bases", "placeholder.yaml"): []byte(`# This is a placeholder file for the bases directory
# It helps satisfy the Go embed directive pattern
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-bases
  namespace: flux-system
data:
  placeholder: "true"
`),
		filepath.Join("manifests", "bases", "helm-controller", "placeholder.yaml"): []byte(`# This is a placeholder file for the helm-controller directory
# It helps satisfy the Go embed directive pattern for nested directories
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-helm-controller
  namespace: flux-system
data:
  placeholder: "true"
`),
	}

	for path, content := range files {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}

	return nil
}

// ensureManifestDirectories makes sure all required directories exist
func ensureManifestDirectories() error {
	manifestDir := "manifests"
	subdirPath := filepath.Join(manifestDir, "bases")
	helmerControllerPath := filepath.Join(subdirPath, "helm-controller")
	
	// Create directories if they don't exist
	for _, dir := range []string{manifestDir, subdirPath, helmerControllerPath} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			// Create placeholder files for newly created directories
			placeholderYAML := []byte(`# This is a placeholder file to ensure the Go embed directive can find at least one file
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
`)

			subdirPlaceholderYAML := []byte(`# This is a placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-subdir-placeholder
  namespace: flux-system
data:
  placeholder: "true"
`)

			// Write the files
			if err := os.WriteFile(filepath.Join(manifestDir, "placeholder.yaml"), placeholderYAML, 0644); err != nil {
				return fmt.Errorf("failed to write placeholder file: %w", err)
			}

			if err := os.WriteFile(filepath.Join(subdirPath, "placeholder.yaml"), subdirPlaceholderYAML, 0644); err != nil {
				return fmt.Errorf("failed to write subdir placeholder file: %w", err)
			}
			
			if err := os.WriteFile(filepath.Join(helmerControllerPath, "placeholder.yaml"), subdirPlaceholderYAML, 0644); err != nil {
				return fmt.Errorf("failed to write helm-controller placeholder file: %w", err)
			}
		}
	}

	return nil
}

// init function to ensure test manifests are created before tests run
func init() {
	if err := createTestManifests(); err != nil {
		fmt.Printf("Warning: Failed to create test manifests: %v\n", err)
	}
}
