# Testing with LocalStack

LocalStack provides a local AWS cloud stack for testing without real AWS credentials. This is ideal for development and CI/CD pipelines.

## Overview

LocalStack emulates AWS services locally, including S3. s3fs-go can connect to LocalStack using the `-endpoint` flag to test S3 operations without requiring real AWS credentials.

## Prerequisites

- Docker and Docker Compose installed
- LocalStack will be started automatically via Docker Compose

## Quick Start

### 1. Start LocalStack

```bash
docker-compose -f docker-compose.localstack.yml up -d
```

### 2. Wait for LocalStack to be Ready

```bash
# Check health endpoint
curl http://localhost:4566/_localstack/health

# Should return JSON with "s3": "available"
```

### 3. Run Tests

```bash
# Run LocalStack integration tests
go test -v ./internal/s3client -run TestLocalStack

# Or use the fast test runner
./run_tests_fast.sh -run TestLocalStack
```

## Running LocalStack Tests

### Using Go Test

```bash
# Run all LocalStack tests
go test -v ./internal/s3client -run TestLocalStack

# Run specific test
go test -v ./internal/s3client -run TestLocalStackPutGet
```

### Using Docker Container

```bash
# Using the persistent test container
docker exec s3fs-go-test go test -v ./internal/s3client -run TestLocalStack
```

## LocalStack Integration Tests

The LocalStack integration tests (`internal/s3client/localstack_integration_test.go`) perform real S3 operations without mocks:

- **TestLocalStackPutGet**: Tests putting and getting objects
- **TestLocalStackListObjects**: Tests listing objects with prefixes
- **TestLocalStackDeleteObject**: Tests deleting objects
- **TestLocalStackGetObjectRange**: Tests range requests
- **TestLocalStackHeadObject**: Tests metadata retrieval
- **TestLocalStackHeadObjectSize**: Tests size retrieval
- **TestLocalStackIntegration**: Comprehensive end-to-end test

These tests automatically:
- Check if LocalStack is running (skip if not available)
- Create test buckets as needed
- Clean up test data after execution

## Manual Testing with LocalStack

### 1. Start LocalStack

```bash
docker-compose -f docker-compose.localstack.yml up -d
```

### 2. Create a Test Bucket

```bash
# Using AWS CLI (if installed)
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
AWS_DEFAULT_REGION=us-east-1 \
aws --endpoint-url=http://localhost:4566 s3 mb s3://test-bucket

# Or using curl
curl -X PUT http://localhost:4566/test-bucket
```

### 3. Set LocalStack Credentials

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
```

### 4. Mount with LocalStack

```bash
./s3fs -bucket test-bucket \
       -mountpoint /mnt/s3 \
       -region us-east-1 \
       -endpoint http://localhost:4566
```

### 5. Use the Mounted Filesystem

```bash
# List files
ls /mnt/s3

# Create a file
echo "Hello LocalStack" > /mnt/s3/test.txt

# Read a file
cat /mnt/s3/test.txt

# List again
ls -la /mnt/s3
```

### 6. Unmount

```bash
fusermount -u /mnt/s3
```

### 7. Stop LocalStack

```bash
docker-compose -f docker-compose.localstack.yml down
```

## Using the Test Script

A convenience script is provided for full LocalStack testing:

```bash
chmod +x test-localstack.sh
./test-localstack.sh
```

This script will:
1. Start LocalStack
2. Create a test bucket
3. Upload test files
4. Build s3fs binary
5. Mount the filesystem
6. Perform basic filesystem operations
7. Clean up and stop LocalStack

**Note**: The test script requires FUSE support (Linux/macOS). On Windows, use WSL or run only the S3 client tests.

## LocalStack Configuration

The LocalStack configuration is in `docker-compose.localstack.yml`:

```yaml
services:
  localstack:
    image: localstack/localstack:latest
    container_name: s3fs-localstack
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3
      - DEBUG=1
    volumes:
      - "./localstack-data:/var/lib/localstack"
```

## Troubleshooting

### LocalStack Not Starting

```bash
# Check logs
docker-compose -f docker-compose.localstack.yml logs

# Restart LocalStack
docker-compose -f docker-compose.localstack.yml restart
```

### Tests Skipping

If tests are being skipped, check:

1. LocalStack is running:
   ```bash
   docker ps | grep localstack
   ```

2. Health endpoint responds:
   ```bash
   curl http://localhost:4566/_localstack/health
   ```

3. Port 4566 is not in use by another service

### Bucket Creation Fails

LocalStack may need a moment to initialize. Wait a few seconds after starting and try again.

## Benefits of LocalStack Testing

- **No AWS Costs**: Test without using real AWS resources
- **Fast**: Local testing is faster than cloud testing
- **Isolated**: Tests don't affect production data
- **CI/CD Friendly**: Easy to integrate into CI/CD pipelines
- **Offline Development**: Work without internet connection

## Limitations

- Not all AWS S3 features are supported
- Performance characteristics differ from real S3
- Some edge cases may behave differently

For production-like testing, use real AWS S3 or Cloudflare R2.
