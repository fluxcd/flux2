#!/bin/bash
set -e

echo "Creating manifest files in cmd/flux directory for go:embed..."

# Create manifests directory inside cmd/flux
mkdir -p cmd/flux/manifests
mkdir -p cmd/flux/manifests/subdir

# Create a placeholder.yaml file in cmd/flux/manifests directory
cat > cmd/flux/manifests/placeholder.yaml << 'EOF'
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
cat > cmd/flux/manifests/subdir/placeholder.yaml << 'EOF'
# This is a placeholder file in a subdirectory for the go:embed directive
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-subdir-placeholder
  namespace: flux-system
data:
  placeholder: "true"
EOF

echo "Manifest files created successfully in cmd/flux directory."
echo "Files in cmd/flux/manifests directory:"
find cmd/flux/manifests -type f | sort

# Remove the problematic fix-embed-issue.go file that has syntax errors
if [ -f "fix-embed-issue.go" ]; then
    echo "Removing problematic fix-embed-issue.go file..."
    rm fix-embed-issue.go
fi

echo "Now try running: go test ./..."
