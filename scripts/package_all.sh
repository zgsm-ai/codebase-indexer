#!/bin/bash

# Multi-platform build script wrapper
# Calls build.sh for all supported platform/arch combinations

# Function to display usage
usage() {
  echo "Usage: $0 VERSION"
  echo "VERSION: Version string for the build (required)"
  exit 1
}

# Check if version is provided
if [ -z "$1" ]; then
  usage
  exit 1
fi

VERSION="$1"

echo "Starting multi-platform build for version: $VERSION"
echo ""

# Supported platforms and architectures
PLATFORMS=("linux" "windows" "darwin")
ARCHITECTURES=("amd64" "arm64")

# Build all combinations
for os in "${PLATFORMS[@]}"; do
  for arch in "${ARCHITECTURES[@]}"; do
    echo "==== Building for $os/$arch ===="
    ./build.sh "$os" "$arch" "$VERSION"
    if [ $? -ne 0 ]; then
      echo "Build failed for $os/$arch"
      exit 1
    fi
    echo ""
  done
done

echo "All builds completed successfully!"
echo "Binaries can be found in bin/$VERSION directory"