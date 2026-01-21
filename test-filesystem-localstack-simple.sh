#!/bin/sh
# Simplified filesystem test script for s3fs-go with LocalStack
# This version assumes LocalStack is already running
# Run this on Linux/macOS host system (not inside Docker)

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

echo "${GREEN}=== s3fs-go LocalStack Filesystem Test ===${NC}"
echo ""

# Check if LocalStack is running
echo "${YELLOW}Checking LocalStack availability...${NC}"
if ! curl -s ${ENDPOINT}/_localstack/health | grep -q "\"s3\": \"available\""; then
    echo "${RED}Error: LocalStack is not running or S3 service is not available${NC}"
    echo "Start LocalStack with: docker-compose -f docker-compose.localstack.yml up -d"
    exit 1
fi
echo "${GREEN}LocalStack is running ✓${NC}"
echo ""

# Create bucket using AWS CLI (if available) or curl
echo "${YELLOW}Creating bucket: ${BUCKET_NAME}${NC}"
if command -v aws > /dev/null 2>&1; then
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
    AWS_DEFAULT_REGION=${REGION} \
    aws --endpoint-url=${ENDPOINT} s3 mb s3://${BUCKET_NAME} 2>/dev/null || echo "Bucket may already exist"
else
    # Use curl to create bucket
    curl -X PUT ${ENDPOINT}/${BUCKET_NAME} 2>/dev/null || echo "Bucket may already exist"
fi

# Upload a test file
echo "${YELLOW}Uploading test file...${NC}"
if command -v aws > /dev/null 2>&1; then
    echo "Hello from LocalStack!" | \
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
    AWS_DEFAULT_REGION=${REGION} \
    aws --endpoint-url=${ENDPOINT} s3 cp - s3://${BUCKET_NAME}/test.txt 2>/dev/null || true
else
    echo "Hello from LocalStack!" | \
    curl -X PUT ${ENDPOINT}/${BUCKET_NAME}/test.txt --data-binary @- 2>/dev/null || true
fi

# Build s3fs binary
echo "${YELLOW}Building s3fs binary...${NC}"
if ! go build -o s3fs ./cmd/s3fs; then
    echo "${RED}Failed to build s3fs${NC}"
    exit 1
fi

# Create mount point
echo "${YELLOW}Creating mount point: ${MOUNTPOINT}${NC}"
mkdir -p ${MOUNTPOINT}

# Check if already mounted
if mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo "${YELLOW}Unmounting existing mount...${NC}"
    fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
fi

# Set credentials for LocalStack (dummy credentials)
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Mount filesystem
echo "${YELLOW}Mounting s3fs...${NC}"
echo "Command: ./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT}"
./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT} &
S3FS_PID=$!

# Wait for mount
sleep 3

# Check if mounted
if ! mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo "${RED}Failed to mount filesystem${NC}"
    kill $S3FS_PID 2>/dev/null || true
    exit 1
fi

echo "${GREEN}Filesystem mounted successfully!${NC}"
echo ""

# Test operations
echo "${YELLOW}=== Testing Filesystem Operations ===${NC}"

# List directory
echo "${YELLOW}Listing directory:${NC}"
ls -la ${MOUNTPOINT} || true

# Read file
echo "${YELLOW}Reading test file:${NC}"
cat ${MOUNTPOINT}/test.txt || true

# Create new file
echo "${YELLOW}Creating new file:${NC}"
echo "Test content" > ${MOUNTPOINT}/newfile.txt || true
cat ${MOUNTPOINT}/newfile.txt || true

# List again
echo "${YELLOW}Listing directory again:${NC}"
ls -la ${MOUNTPOINT} || true

# Cleanup
echo ""
echo "${YELLOW}=== Cleanup ===${NC}"
echo "Unmounting filesystem..."
fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
sleep 1

# Kill s3fs if still running
kill $S3FS_PID 2>/dev/null || true

echo "${GREEN}Test completed!${NC}"
