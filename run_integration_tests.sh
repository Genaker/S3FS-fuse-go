#!/bin/bash
# Run integration tests with LocalStack or production S3/R2

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if LocalStack is required (default)
PROVIDER="${S3_PROVIDER:-localstack}"

if [ "$PROVIDER" = "localstack" ]; then
    echo "Checking LocalStack availability..."
    
    # Check if LocalStack is running
    if ! curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1; then
        echo "ERROR: LocalStack is not running!"
        echo ""
        echo "Start LocalStack with:"
        echo "  docker-compose -f docker-compose.localstack.yml up -d"
        echo ""
        echo "Or set S3_PROVIDER=s3 or S3_PROVIDER=r2 to use production services"
        exit 1
    fi
    
    echo "LocalStack is running ✓"
fi

# Run integration tests
echo ""
echo "Running integration tests with provider: $PROVIDER"
echo ""

# Export provider for tests
export S3_PROVIDER="$PROVIDER"

# Run tests with integration build tag
go test -tags=integration ./tests/... -v "$@"
