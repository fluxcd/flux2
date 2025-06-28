#!/bin/bash
# This script creates the necessary manifests for go:embed to work

set -e

echo "Creating manifest files for testing..."

# Create manifests in the project root
mkdir -p manifests
mkdir -p manifests/subdir

# Create a placeholder.yaml file in the manifests directory
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

# Create a second placeholder in the manifests directory
cat > manifests/dummy.yaml << 'EOF'
# This is another placeholder file for the go:embed directive
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-dummy
  namespace: flux-system
data:
  placeholder: "true"
EOF

# Create a placeholder in the subdir
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

# Create manifests in the cmd/flux directory
mkdir -p cmd/flux/manifests
mkdir -p cmd/flux/manifests/subdir

# Create a placeholder.yaml file in the cmd/flux/manifests directory
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

# Create a second placeholder in the cmd/flux/manifests directory
cat > cmd/flux/manifests/dummy.yaml << 'EOF'
# This is another placeholder file for the go:embed directive
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-dummy
  namespace: flux-system
data:
  placeholder: "true"
EOF

# Create a placeholder in the cmd/flux/manifests/subdir
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

echo "Manifest files created successfully."
echo "Files in manifests directory:"
find manifests -type f | sort
echo "Files in cmd/flux/manifests directory:"
find cmd/flux/manifests -type f | sort
