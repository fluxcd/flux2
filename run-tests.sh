#!/bin/bash
set -e

echo "Setting up test environment for flux2..."

# Ensure the required directories exist
mkdir -p manifests
mkdir -p manifests/subdir

# Create placeholder YAML files
echo "Creating placeholder YAML files..."

cat > manifests/placeholder.yaml << 'EOF'
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

cat > manifests/subdir/placeholder.yaml << 'EOF'
# This is a placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-subdir-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

echo "Running tests..."
go test ./...
