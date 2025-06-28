package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Create the manifests directory structure
	manifestsDir := "manifests"
	os.MkdirAll(manifestsDir, 0755)
	os.MkdirAll(filepath.Join(manifestsDir, "subdir"), 0755)

	// Create a placeholder manifest file at the root
	placeholderPath := filepath.Join(manifestsDir, "placeholder.yaml")
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
	if err := os.WriteFile(placeholderPath, []byte(placeholderContent), 0644); err != nil {
		fmt.Printf("Error creating placeholder.yaml: %v\n", err)
		os.Exit(1)
	}

	// Create a placeholder in a subdirectory
	subDirPlaceholderPath := filepath.Join(manifestsDir, "subdir", "another-placeholder.yaml")
	subDirPlaceholderContent := `# This is another placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-subdir
  namespace: flux-system
data:
  placeholder: "true"
`
	if err := os.WriteFile(subDirPlaceholderPath, []byte(subDirPlaceholderContent), 0644); err != nil {
		fmt.Printf("Error creating another-placeholder.yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Created placeholder files for Go embed directive")
}
