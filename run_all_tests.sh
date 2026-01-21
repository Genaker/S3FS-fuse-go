#!/bin/bash
# Run all tests (unit + integration)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Running Unit Tests ==="
go test ./internal/credentials/... ./internal/s3client/... ./internal/fuse/... -v

echo ""
echo "=== Running Integration Tests ==="
./run_integration_tests.sh
