#!/bin/bash
set -e

echo "Creating manifest files for go:embed..."

# Change to the root of the project directory
cd "$(git rev-parse --show-toplevel)" || cd "/home/calelin/flux2"

# Ensure manifests directory exists
mkdir -p manifests
mkdir -p manifests/subdir

# Create a placeholder.yaml file in manifests directory
cat > manifests/placeholder.yaml << 'EOF'
# This is a placeholder file for the go:embed directive
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

# Create another placeholder in a subdirectory
cat > manifests/subdir/placeholder.yaml << 'EOF'
# This is a placeholder file in a subdirectory for the go:embed directive
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-subdir-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

echo "Manifest files created successfully."
echo "Files in manifests directory:"
find manifests -type f | sort
