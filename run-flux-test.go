package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Create the manifests directory structure
	manifestsDir := "manifests"
	subDir := filepath.Join(manifestsDir, "subdir")
	
	fmt.Println("Setting up test environment...")
	
	// Create directories
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		fmt.Printf("Error creating manifests directory: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.MkdirAll(subDir, 0755); err != nil {
		fmt.Printf("Error creating manifests/subdir directory: %v\n", err)
		os.Exit(1)
	}
	
	// Create placeholder files
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
	
	subdirContent := `# This is a placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-subdir-placeholder
  namespace: flux-system
data:
  placeholder: "true"
`
	
	if err := os.WriteFile(filepath.Join(manifestsDir, "placeholder.yaml"), []byte(placeholderContent), 0644); err != nil {
		fmt.Printf("Error writing placeholder.yaml: %v\n", err)
		os.Exit(1)
	}
	
	if err := os.WriteFile(filepath.Join(subDir, "placeholder.yaml"), []byte(subdirContent), 0644); err != nil {
		fmt.Printf("Error writing subdir/placeholder.yaml: %v\n", err)
		os.Exit(1)
	}
	
	// Check the file content and structure
	fmt.Println("Created placeholder files. Directory structure:")
	cmd := exec.Command("find", "manifests", "-type", "f", "-name", "*.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error listing files: %v\n", err)
	}
	
	// Run the tests
	fmt.Println("\nRunning tests...")
	testCmd := exec.Command("go", "test", "./...")
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		fmt.Printf("Tests failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Tests completed successfully!")
}
