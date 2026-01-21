#!/bin/bash
# Test coverage statistics script

set -e

echo "=== s3fs-go Test Coverage Statistics ==="
echo ""

# Run tests with coverage for each package
echo "Running tests with coverage..."
go test ./... -coverprofile=coverage.out -cover

echo ""
echo "=== Coverage by Package ==="
go test ./... -cover | grep -E "(ok|FAIL|coverage)"

echo ""
echo "=== Detailed Coverage Report ==="
go tool cover -func=coverage.out

echo ""
echo "=== Coverage Summary ==="
go tool cover -func=coverage.out | grep "total:" | awk '{print "Total Coverage: " $3}'

echo ""
echo "=== Generating HTML Coverage Report ==="
go tool cover -html=coverage.out -o coverage.html
echo "HTML report generated: coverage.html"

echo ""
echo "=== Coverage by Package Details ==="
echo ""
echo "Credentials Package:"
go test ./internal/credentials -cover | grep coverage

echo ""
echo "S3 Client Package:"
go test ./internal/s3client -cover | grep coverage

echo ""
echo "FUSE Package:"
go test ./internal/fuse -cover | grep coverage

echo ""
echo "=== Test Counts ==="
echo ""
echo "Credentials tests:"
go test ./internal/credentials -v 2>&1 | grep -c "RUN\|PASS\|FAIL\|SKIP" || echo "0"

echo ""
echo "S3 Client tests:"
go test ./internal/s3client -v 2>&1 | grep -c "RUN\|PASS\|FAIL\|SKIP" || echo "0"

echo ""
echo "FUSE tests:"
go test ./internal/fuse -v 2>&1 | grep -c "RUN\|PASS\|FAIL\|SKIP" || echo "0"
