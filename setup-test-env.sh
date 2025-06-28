#!/bin/bash
set -e

echo "Setting up test environment for flux2..."

# Create the manifests directory
mkdir -p manifests

# Create placeholder.yaml file
cat > manifests/placeholder.yaml << EOF
# This is a placeholder file to ensure the Go embed directive can find at least one file
# It will be replaced by actual manifests when bundle.sh is run successfully
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

# Create subdirectories for the pattern matching to work
mkdir -p manifests/subdir

# Create another YAML file in a subdirectory
cat > manifests/subdir/another-placeholder.yaml << EOF
# This is another placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-subdir
  namespace: flux-system
data:
  placeholder: "true"
EOF

echo "Test environment set up successfully!"
echo "You can now run 'go test ./...' to run the tests"
