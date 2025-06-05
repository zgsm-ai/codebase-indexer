#!/bin/bash

# Go Build Script for Specific Platform

# Function to display usage
usage() {
  echo "Usage: $0 <GOOS> <GOARCH> <VERSION>"
  echo "Example: $0 linux amd64 v1.0.0"
  echo "Example: $0 windows amd64 1.0.0-beta"
  echo "Example: $0 darwin arm64 latest"
  echo ""
  echo "Parameters:"
  echo "  <GOOS>      Target operating system (e.g., linux, windows, darwin)"
  echo "  <GOARCH>    Target architecture (e.g., amd64, 386, arm64)"
  echo "  <VERSION>   Version string for the build (e.g., v1.0.0, 0.1.x)"
  exit 1
}

# Check if all three arguments are provided
if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
  usage
fi

TARGET_OS="$1"
TARGET_ARCH="$2"
APP_VERSION="$3"

# Get the directory of the script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# Navigate to the project root directory (assuming scripts directory is one level down from root)
PROJECT_ROOT="$(realpath "$SCRIPT_DIR/..")"
cd "$PROJECT_ROOT" || exit

# Define the output directory for builds
OUTPUT_DIR="bin" # Output directory relative to project root
mkdir -p "$PROJECT_ROOT/$OUTPUT_DIR"

# Define the main Go package path
MAIN_PACKAGE_PATH="cmd/main.go" # Adjust if your main package is elsewhere

# Common build flags to reduce binary size
# We can also embed the version into the binary itself.
# For this to work, you'd need a variable in your Go main package, e.g.:
# var version string
# Then uncomment and adjust the line below:
# LD_FLAGS="-s -w -X main.version=$APP_VERSION"
LD_FLAGS="-s -w -X main.version=$APP_VERSION"

# Determine output filename
OUTPUT_FILENAME="codebase-syncer_${APP_VERSION}_${TARGET_OS}_${TARGET_ARCH}"
if [ "$TARGET_OS" = "windows" ]; then
  OUTPUT_FILENAME="${OUTPUT_FILENAME}.exe"
fi

echo "Starting Go build process for Version: $APP_VERSION, OS: $TARGET_OS, Arch: $TARGET_ARCH..."

# Set environment variables and build
GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -ldflags="$LD_FLAGS" -o "$PROJECT_ROOT/$OUTPUT_DIR/$OUTPUT_FILENAME" "$MAIN_PACKAGE_PATH"

if [ $? -eq 0 ]; then
  echo "Build successful!"
  echo "Executable created at: $PROJECT_ROOT/$OUTPUT_DIR/$OUTPUT_FILENAME"
else
  echo "Build failed."
  exit 1
fi

echo "Build process completed."