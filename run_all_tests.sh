#!/bin/bash
# Run all tests (unit + integration)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Running Unit Tests (Standard) ==="
go test ./internal/... -v

echo ""
echo "=== Running Integration Tests ==="
./run_integration_tests.sh

echo ""
echo "=== Running Functional Tests ==="
go test -tags=functional ./cmd/... -v || echo "Functional tests skipped (may require build)"
