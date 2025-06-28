#!/bin/bash

# This script creates placeholder manifest files to satisfy
# the Go embed directive for testing purposes

set -e

# Create the manifests directory if it doesn't exist
mkdir -p manifests

# Create a placeholder manifest file
cat > manifests/placeholder.yaml << EOF
# This is a placeholder manifest file for tests
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

echo "Created placeholder manifest files for testing"
