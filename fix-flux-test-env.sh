#!/bin/bash
set -e

echo "Setting up complete test environment for flux2..."

# Create the manifests directory structure
echo "Creating manifests directory structure..."
mkdir -p manifests
mkdir -p manifests/subdir

# Create placeholder files
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

cat > manifests/subdir/another-placeholder.yaml << 'EOF'
# This is another placeholder file in a subdirectory
apiVersion: v1
kind: ConfigMap
metadata:
  name: flux-placeholder-subdir
  namespace: flux-system
data:
  placeholder: "true"
EOF

# Check for necessary tools
echo "Checking for required tools..."

# Check for kubectl
if ! command -v kubectl &> /dev/null; then
    echo "kubectl is not installed. Installing..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
fi

# Check for kustomize
if ! command -v kustomize &> /dev/null; then
    echo "kustomize is not installed. Installing..."
    curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
    chmod +x kustomize
    sudo mv kustomize /usr/local/bin/
fi

# Check if the Go module exists
echo "Checking Go modules..."
go mod tidy

# Run a test compile to check for issues
echo "Running test compile..."
go build -o /dev/null ./cmd/flux

echo "Test environment setup complete!"
echo "You can now run 'go test ./...' to run the tests"
