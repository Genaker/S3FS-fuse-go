#!/bin/bash
# Test script for s3fs-go with LocalStack

set -e

BUCKET_NAME="test-bucket"
MOUNTPOINT="/tmp/s3fs-mount"
ENDPOINT="http://localhost:4566"
REGION="us-east-1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== s3fs-go LocalStack Test ===${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    exit 1
fi

# Start LocalStack
echo -e "${YELLOW}Starting LocalStack...${NC}"
docker-compose -f docker-compose.localstack.yml up -d

# Wait for LocalStack to be ready
echo -e "${YELLOW}Waiting for LocalStack to be ready...${NC}"
for i in {1..30}; do
    if curl -s http://localhost:4566/_localstack/health | grep -q "\"s3\": \"available\""; then
        echo -e "${GREEN}LocalStack is ready!${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}LocalStack failed to start${NC}"
        docker-compose -f docker-compose.localstack.yml logs
        exit 1
    fi
    sleep 2
done

# Create bucket using AWS CLI (if available) or curl
echo -e "${YELLOW}Creating bucket: ${BUCKET_NAME}${NC}"
if command -v aws &> /dev/null; then
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
    AWS_DEFAULT_REGION=${REGION} \
    aws --endpoint-url=${ENDPOINT} s3 mb s3://${BUCKET_NAME} || true
else
    # Use curl to create bucket
    curl -X PUT ${ENDPOINT}/${BUCKET_NAME} || true
fi

# Upload a test file
echo -e "${YELLOW}Uploading test file...${NC}"
if command -v aws &> /dev/null; then
    echo "Hello from LocalStack!" | \
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
    AWS_DEFAULT_REGION=${REGION} \
    aws --endpoint-url=${ENDPOINT} s3 cp - s3://${BUCKET_NAME}/test.txt
else
    echo "Hello from LocalStack!" | \
    curl -X PUT ${ENDPOINT}/${BUCKET_NAME}/test.txt --data-binary @-
fi

# Build s3fs binary
echo -e "${YELLOW}Building s3fs binary...${NC}"
if ! go build -o s3fs ./cmd/s3fs; then
    echo -e "${RED}Failed to build s3fs${NC}"
    exit 1
fi

# Create mount point
echo -e "${YELLOW}Creating mount point: ${MOUNTPOINT}${NC}"
mkdir -p ${MOUNTPOINT}

# Check if already mounted
if mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo -e "${YELLOW}Unmounting existing mount...${NC}"
    fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
fi

# Set credentials for LocalStack (dummy credentials)
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Mount filesystem
echo -e "${YELLOW}Mounting s3fs...${NC}"
echo "Command: ./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT}"
./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT} &
S3FS_PID=$!

# Wait for mount
sleep 3

# Check if mounted
if ! mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo -e "${RED}Failed to mount filesystem${NC}"
    kill $S3FS_PID 2>/dev/null || true
    exit 1
fi

echo -e "${GREEN}Filesystem mounted successfully!${NC}"
echo ""

# Test operations
echo -e "${YELLOW}=== Testing Filesystem Operations ===${NC}"

# List directory
echo -e "${YELLOW}Listing directory:${NC}"
ls -la ${MOUNTPOINT} || true

# Read file
echo -e "${YELLOW}Reading test file:${NC}"
cat ${MOUNTPOINT}/test.txt || true

# Create new file
echo -e "${YELLOW}Creating new file:${NC}"
echo "Test content" > ${MOUNTPOINT}/newfile.txt || true
cat ${MOUNTPOINT}/newfile.txt || true

# List again
echo -e "${YELLOW}Listing directory again:${NC}"
ls -la ${MOUNTPOINT} || true

# Cleanup
echo ""
echo -e "${YELLOW}=== Cleanup ===${NC}"
echo "Unmounting filesystem..."
fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
sleep 1

# Kill s3fs if still running
kill $S3FS_PID 2>/dev/null || true

# Stop LocalStack
echo "Stopping LocalStack..."
docker-compose -f docker-compose.localstack.yml down

echo -e "${GREEN}Test completed!${NC}"
