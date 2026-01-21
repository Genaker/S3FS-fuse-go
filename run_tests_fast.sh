#!/bin/bash
# Fast test runner using persistent Docker container

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

CONTAINER_NAME="s3fs-go-test"

# Check if container exists
if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Creating persistent test container..."
    docker run -d --name ${CONTAINER_NAME} \
        --network host \
        -v "$(pwd):/app" \
        -w /app \
        golang:1.21-alpine \
        sh -c "apk add --no-cache git curl && tail -f /dev/null"
    echo "Waiting for container to be ready..."
    sleep 2
fi

# Start container if stopped
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Starting container..."
    docker start ${CONTAINER_NAME}
    sleep 1
fi

# Check if running integration tests
if [[ "$*" == *"-tags=integration"* ]] || [[ "$*" == *"integration"* ]]; then
    PROVIDER="${S3_PROVIDER:-localstack}"
    
    if [ "$PROVIDER" = "localstack" ]; then
        echo "Checking LocalStack availability..."
        if ! docker exec ${CONTAINER_NAME} sh -c "curl -s http://localhost:4566/_localstack/health > /dev/null 2>&1"; then
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
    
    export S3_PROVIDER="$PROVIDER"
fi

# Run tests
echo "Running tests..."
docker exec -e S3_PROVIDER="${S3_PROVIDER:-localstack}" ${CONTAINER_NAME} go test ./... "$@"
