#!/bin/bash
set -e

# This script checks if the placeholder YAML files exist
# and creates them if they don't

echo "Checking for placeholder YAML files..."

# Ensure the required directories exist
mkdir -p manifests
mkdir -p manifests/subdir

# Create placeholder YAML files if they don't exist
if [ ! -f "manifests/placeholder.yaml" ]; then
  echo "Creating manifests/placeholder.yaml..."
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
fi

if [ ! -f "manifests/subdir/placeholder.yaml" ]; then
  echo "Creating manifests/subdir/placeholder.yaml..."
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
fi

echo "Placeholder YAML files check complete."
