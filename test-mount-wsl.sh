#!/bin/bash
# Simple WSL test script for filesystem mounting

set -e

BUCKET_NAME="test-bucket"
MOUNTPOINT="/tmp/s3fs-mount"
ENDPOINT="http://localhost:4566"
REGION="us-east-1"

echo "=== s3fs-go WSL Filesystem Test ==="
echo ""

# Check LocalStack
echo "Checking LocalStack..."
HEALTH=$(curl -s ${ENDPOINT}/_localstack/health 2>/dev/null)
if echo "$HEALTH" | grep -q "\"s3\": \"running\"" || echo "$HEALTH" | grep -q "\"s3\": \"available\""; then
    echo "LocalStack is running ✓"
else
    echo "ERROR: LocalStack S3 service not available"
    echo "Health check response: $HEALTH"
    exit 1
fi
echo ""

# Create bucket
echo "Creating bucket..."
curl -X PUT ${ENDPOINT}/${BUCKET_NAME} 2>/dev/null || echo "Bucket may already exist"

# Upload test file (overwrite if exists)
echo "Uploading test file..."
echo "Hello from LocalStack!" | curl -X PUT ${ENDPOINT}/${BUCKET_NAME}/test.txt --data-binary @- 2>/dev/null || {
    echo "WARNING: Failed to upload test file via curl, continuing anyway..."
}

# Create mount point
echo "Creating mount point: ${MOUNTPOINT}"
mkdir -p ${MOUNTPOINT}

# Cleanup: Unmount if already mounted and kill any existing s3fs processes
if mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo "Unmounting existing mount..."
    fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
    sleep 1
fi

# Kill any existing s3fs processes for this bucket
pkill -f "s3fs.*${BUCKET_NAME}" 2>/dev/null || true
sleep 1

# Set credentials
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Mount filesystem
echo "Mounting s3fs..."
echo "Command: ./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT}"
./s3fs -bucket ${BUCKET_NAME} -mountpoint ${MOUNTPOINT} -region ${REGION} -endpoint ${ENDPOINT} > /tmp/s3fs.log 2>&1 &
S3FS_PID=$!

# Wait for mount
sleep 4

# Check if mounted
if mountpoint -q ${MOUNTPOINT} 2>/dev/null; then
    echo "SUCCESS: Filesystem mounted!"
    echo ""
    
    # Test operations
    echo "=== Testing Filesystem Operations ==="
    echo ""
    
    echo "Listing directory:"
    ls -la ${MOUNTPOINT} 2>/dev/null || ls -la ${MOUNTPOINT}/* 2>/dev/null || echo "Directory listing completed"
    echo ""
    
    echo "Reading test file:"
    if [ -f ${MOUNTPOINT}/test.txt ]; then
        cat ${MOUNTPOINT}/test.txt
        echo ""
    else
        echo "ERROR: test.txt not found!"
        exit 1
    fi
    
    echo "Creating new file:"
    TEST_CONTENT="Test content from WSL - $(date +%s)"
    echo "${TEST_CONTENT}" > ${MOUNTPOINT}/newfile.txt
    if [ -f ${MOUNTPOINT}/newfile.txt ]; then
        READ_CONTENT=$(cat ${MOUNTPOINT}/newfile.txt)
        echo "${READ_CONTENT}"
        if [ "${READ_CONTENT}" != "${TEST_CONTENT}" ]; then
            echo "ERROR: File content mismatch!"
            exit 1
        fi
        echo "✓ File write/read verified"
    else
        echo "ERROR: Failed to create newfile.txt!"
        exit 1
    fi
    echo ""
    
    echo "Listing directory again:"
    ls -la ${MOUNTPOINT} 2>/dev/null || ls -la ${MOUNTPOINT}/* 2>/dev/null || echo "Directory listing completed"
    echo ""
    
    # Cleanup
    echo "=== Cleanup ==="
    echo "Unmounting filesystem..."
    fusermount -u ${MOUNTPOINT} 2>/dev/null || umount ${MOUNTPOINT} 2>/dev/null || true
    sleep 1
    
    kill $S3FS_PID 2>/dev/null || true
    
    echo "Test completed successfully!"
else
    echo "FAILED to mount filesystem"
    echo "Logs:"
    cat /tmp/s3fs.log
    kill $S3FS_PID 2>/dev/null || true
    exit 1
fi
