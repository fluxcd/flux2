package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	// Set up test environment
	setupTestEnvironment()

	// Run tests
	exitCode := m.Run()

	// Exit with the same code
	os.Exit(exitCode)
}

func setupTestEnvironment() {
	// Create the manifests directory structure
	manifestsDir := filepath.Join("..", "..", "manifests")
	os.MkdirAll(manifestsDir, 0755)

	// Create a placeholder manifest file if it doesn't exist
	placeholderPath := filepath.Join(manifestsDir, "placeholder.yaml")
	if _, err := os.Stat(placeholderPath); os.IsNotExist(err) {
		placeholderContent := `# This is a placeholder file to ensure the Go embed directive can find at least one file
# It will be replaced by actual manifests when bundle.sh is run successfully
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
`
		os.WriteFile(placeholderPath, []byte(placeholderContent), 0644)
	}
}
