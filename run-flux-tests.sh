#!/bin/bash
set -e

# Source directory
SOURCE_DIR=$(pwd)

# Run the fix script first
echo "Setting up test environment..."
chmod +x ./fix-flux-test-env.sh
./fix-flux-test-env.sh

# Run the tests
echo "Running tests..."
go test ./...

# If we get here, the tests passed
echo "Tests completed successfully!"
