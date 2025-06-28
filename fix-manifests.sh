#!/bin/bash
set -e

echo "Creating placeholder YAML files for Go embed directive..."

# Ensure manifests directory exists
mkdir -p manifests
mkdir -p manifests/subdir

# Create the main placeholder.yaml
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

# Create a placeholder in a subdirectory
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

echo "Placeholder YAML files created successfully."
echo "You can now run the tests with: go test ./..."
