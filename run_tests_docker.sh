#!/bin/bash
# Run tests using Docker

set -e

echo "=== Running s3fs-go tests in Docker ==="
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    echo "Please install Docker from https://www.docker.com/get-started"
    exit 1
fi

echo "Building Docker image..."
docker build -t s3fs-go-test .

echo ""
echo "Running tests..."
docker run --rm s3fs-go-test

echo ""
echo "Tests completed!"
