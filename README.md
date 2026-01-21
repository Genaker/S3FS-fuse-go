# s3fs-go

FUSE-based filesystem for S3-compatible storage.

## Overview

s3fs-go is a FUSE-based filesystem that allows you to mount an S3-compatible bucket as a local filesystem.

**Supported Services:**
- Amazon S3
- Cloudflare R2
- LocalStack (for testing)
- Any S3-compatible service

## Prerequisites

- **Go 1.21 or later** - [Download Go](https://golang.org/dl/)
- **FUSE** - Required for mounting filesystems
  - **Linux**: Install `libfuse-dev` or `fuse-devel`
  - **macOS**: Install `osxfuse` via Homebrew: `brew install osxfuse`
  - **Windows**: Not directly supported (requires WSL or similar)

- **AWS Credentials** - One of the following:
  - AWS credentials file (`~/.aws/credentials`)
  - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
  - Passwd file (see Configuration section)

## Building

```bash
# Build the s3fs binary
go build -o s3fs ./cmd/s3fs

# Or build with specific output name
go build -o s3fs-go ./cmd/s3fs
```

## Running

### Basic Usage

```bash
# Mount an S3 bucket
./s3fs -bucket my-bucket-name -mountpoint /mnt/s3 -region us-east-1

# With passwd file
./s3fs -bucket my-bucket-name -mountpoint /mnt/s3 -passwd_file ~/.passwd-s3fs

# Using environment variables for credentials
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
./s3fs -bucket my-bucket-name -mountpoint /mnt/s3
```

### Command Line Options

- `-bucket`: S3 bucket name (required)
- `-mountpoint`: Mount point directory (required)
- `-region`: AWS region (default: `us-east-1`)
- `-endpoint`: S3 endpoint URL (for LocalStack or other S3-compatible services, optional)
- `-passwd_file`: Path to passwd file containing credentials (optional)
- `-enable_file_lock`: Enable file-level advisory locking for stricter coordination (default: `false`, uses entity-level locking)

### Example

```bash
# Create mount point
sudo mkdir -p /mnt/s3

# Mount bucket
sudo ./s3fs -bucket my-s3-bucket -mountpoint /mnt/s3 -region us-west-2

# Use the mounted filesystem
ls /mnt/s3
cat /mnt/s3/myfile.txt

# Unmount
sudo umount /mnt/s3
```

## Configuration

### Credentials via Passwd File

Create a passwd file with your AWS credentials:

```bash
# Format: ACCESS_KEY_ID:SECRET_ACCESS_KEY
echo "AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" > ~/.passwd-s3fs
chmod 600 ~/.passwd-s3fs
```

Then use it:

```bash
./s3fs -bucket my-bucket -mountpoint /mnt/s3 -passwd_file ~/.passwd-s3fs
```

### Credentials via Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your-access-key-id
export AWS_SECRET_ACCESS_KEY=your-secret-access-key
export AWS_REGION=us-east-1  # Optional, can also use -region flag

./s3fs -bucket my-bucket -mountpoint /mnt/s3
```

### Credentials via AWS Credentials File

If you have AWS CLI configured, the application will automatically use credentials from `~/.aws/credentials`:

```bash
./s3fs -bucket my-bucket -mountpoint /mnt/s3
```

## Using with Cloudflare R2

s3fs-go works seamlessly with Cloudflare R2, an S3-compatible object storage service with no egress fees.

### Quick Start with R2

```bash
# Set R2 credentials
export AWS_ACCESS_KEY_ID=your-r2-access-key-id
export AWS_SECRET_ACCESS_KEY=your-r2-secret-access-key

# Mount R2 bucket
./s3fs -bucket your-r2-bucket \
       -mountpoint /mnt/r2 \
       -region auto \
       -endpoint https://your-account-id.r2.cloudflarestorage.com
```

**Note**: Replace `your-account-id` with your Cloudflare account ID (found in the Cloudflare Dashboard).

For detailed R2 setup instructions, see [Cloudflare R2 Documentation](doc/cloudflare-r2.md).

## Testing

### Test Structure

Tests are organized following Go best practices:

- **Unit Tests**: Located in `internal/*/*_test.go` files (same package)
- **Integration Tests**: Located in `internal/integration/` folder with `//go:build integration` tag

### Run Unit Tests

```bash
# Run all unit tests (excludes integration tests)
go test ./internal/credentials/...
go test ./internal/s3client/...
go test ./internal/fuse/...

# Run with verbose output
go test -v ./internal/...
```

### Run Integration Tests

Integration tests use LocalStack by default, but can be configured to use production S3 or R2.

#### Using LocalStack (Default)

```bash
# Ensure LocalStack is running
docker-compose -f docker-compose.localstack.yml up -d

# Run integration tests (automatically checks LocalStack availability)
chmod +x run_integration_tests.sh
./run_integration_tests.sh

# Or manually
go test -tags=integration ./internal/integration/... -v
```

**Note**: Integration tests will fail if LocalStack is not running (unless using production S3/R2).

#### Using Production S3

```bash
# Set environment variables
export S3_PROVIDER=s3
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_REGION=us-east-1

# Run integration tests
./run_integration_tests.sh
```

#### Using Cloudflare R2

```bash
# Set environment variables
export S3_PROVIDER=r2
export R2_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
export AWS_ACCESS_KEY_ID=your-r2-access-key
export AWS_SECRET_ACCESS_KEY=your-r2-secret-key
export AWS_REGION=auto

# Run integration tests
./run_integration_tests.sh
```

### Run All Tests (Unit + Integration)

```bash
# Run all tests
chmod +x run_all_tests.sh
./run_all_tests.sh

# Or manually
go test ./internal/... -v
go test -tags=integration ./internal/integration/... -v
```

### Run Tests in Docker

#### Fast Test Runner (Recommended)

For faster test execution, use the persistent Docker container:

```bash
# First time: creates and starts a persistent container
chmod +x run_tests_fast.sh
./run_tests_fast.sh

# Subsequent runs: reuses the existing container (much faster)
./run_tests_fast.sh

# Run integration tests (checks LocalStack automatically)
./run_tests_fast.sh -tags=integration ./internal/integration/... -v

# Or manually use the container
docker exec s3fs-go-test go test ./... -v
docker exec s3fs-go-test go test -tags=integration ./internal/integration/... -v
```

The persistent container (`s3fs-go-test`) stays running, eliminating the overhead of starting a new container each time.

#### Standard Docker Test Runner

```bash
# Using Docker Compose
docker-compose up

# Or using the test script
chmod +x run_tests_docker.sh
./run_tests_docker.sh

# Or manually
docker build -t s3fs-go-test .
docker run --rm s3fs-go-test
```

### Testing with LocalStack

LocalStack provides a local AWS cloud stack for testing without real AWS credentials. This is ideal for development and CI/CD pipelines.

#### Prerequisites

- Docker and Docker Compose installed
- LocalStack will be started automatically via Docker Compose

#### Running LocalStack Tests

```bash
# Start LocalStack
docker-compose -f docker-compose.localstack.yml up -d

# Wait for LocalStack to be ready (usually takes 10-20 seconds)
# Check health: curl http://localhost:4566/_localstack/health

# Run integration tests (automatically uses LocalStack)
./run_integration_tests.sh

# Or manually
go test -tags=integration ./internal/integration/... -v
```

#### LocalStack Integration Tests

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

#### Using LocalStack for Manual Testing

```bash
# Start LocalStack
docker-compose -f docker-compose.localstack.yml up -d

# Set LocalStack credentials (dummy values)
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Mount with LocalStack endpoint
./s3fs -bucket test-bucket \
       -mountpoint /mnt/s3 \
       -region us-east-1 \
       -endpoint http://localhost:4566

# Use the mounted filesystem
ls /mnt/s3
echo "Hello" > /mnt/s3/test.txt
cat /mnt/s3/test.txt

# Unmount
fusermount -u /mnt/s3

# Stop LocalStack
docker-compose -f docker-compose.localstack.yml down
```

#### LocalStack Filesystem Test Script

A convenience script is provided for full filesystem testing with LocalStack:

```bash
chmod +x test-filesystem-localstack.sh
./test-filesystem-localstack.sh
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

For detailed LocalStack documentation, see [LocalStack Documentation](doc/localstack.md).

## Project Structure

```
go/
├── cmd/
│   └── s3fs/          # Main application entry point
├── internal/
│   ├── credentials/   # AWS credentials management
│   ├── s3client/      # S3 API client
│   └── fuse/          # FUSE filesystem operations
├── doc/               # Documentation
│   ├── cloudflare-r2.md    # Cloudflare R2 setup guide
│   ├── localstack.md        # LocalStack testing guide
│   └── testing.md           # Testing guide
├── go.mod             # Go module definition
├── Dockerfile         # Docker image for testing
├── docker-compose.yml # Docker Compose configuration
├── docker-compose.localstack.yml # LocalStack Docker Compose config
├── test-filesystem-localstack.sh # LocalStack filesystem mount test script
└── README.md          # This file
```

## Development Status

This is a work in progress, following TDD principles. **Current implementation: Core FUSE operations implemented**

### ✅ Completed
- Credentials package (basic - tests + implementation)
- S3 client package (basic - tests + implementation with AWS SDK)
- FUSE filesystem operations (basic - tests + implementation)
- FUSE wrapper for mounting (basic implementation)
- Multi-part upload support
- File times management
- Extended attributes (xattr)
- Permissions (chmod, chown)

### 🔄 In Progress / Planned
- Caching layer (stat cache, fd cache)
- Advanced FUSE operations (symlink, hardlink, mknod)
- Thread pool for concurrency
- Advanced authentication (IAM roles, EC2 instance metadata)
- Signal handlers

## Troubleshooting

### Permission Denied

If you get permission errors when mounting:

```bash
# Make sure the mount point exists and is accessible
sudo mkdir -p /mnt/s3
sudo chown $USER:$USER /mnt/s3

# On Linux, you may need to add user to fuse group
sudo usermod -aG fuse $USER
# Then logout and login again
```

### FUSE Not Found

If you get errors about FUSE not being available:

**Linux:**
```bash
sudo apt-get install libfuse-dev  # Debian/Ubuntu
sudo yum install fuse-devel      # CentOS/RHEL
```

**macOS:**
```bash
brew install osxfuse
```

### Invalid Credentials

Make sure your credentials are correct:

```bash
# Test credentials with AWS CLI
aws s3 ls s3://my-bucket-name
```

## Write Locking Configuration

s3fs-go supports two locking strategies for concurrent file operations:

### Option 1: Entity-Level Locking (Default)

**Default behavior** - Uses mutex locking at the file entity level:
- Better performance
- Sufficient for most use cases
- Serializes writes to the same file entity
- Works well for single-process FUSE mounts

```bash
# Default - entity-level locking
./s3fs -bucket my-bucket -mountpoint /mnt/s3
```

### Option 2: File-Level Advisory Locking

**Optional** - Provides stricter coordination with file-level locks:
- Stricter serialization of all file operations
- Better coordination for applications that need guaranteed write ordering
- Slightly higher overhead

```bash
# Enable file-level advisory locking
./s3fs -bucket my-bucket -mountpoint /mnt/s3 -enable_file_lock
```

### When to Use Each Option

- **Entity-Level Locking (Default)**: Use for general-purpose file operations, single-process mounts, and when performance is important
- **File-Level Locking**: Use when you need stricter coordination, guaranteed write ordering, or when multiple processes/threads need explicit file-level synchronization

**Note**: Both options work within a single FUSE mount instance. For coordination across multiple mount instances or different machines, consider using S3-native features like ETags or external locking mechanisms.

## License

See the main project LICENSE file.

## Documentation

- [README](README.md) - This file, quick start guide
- [Linux & FUSE Basics](doc/linux-fuse-basics.md) - Simple explanation of Linux, FUSE, and how Go works with them
- [Missing FUSE Features](doc/missing-fuse-features.md) - List of FUSE operations not yet implemented
- [Cloudflare R2 Guide](doc/cloudflare-r2.md) - Using s3fs-go with Cloudflare R2
- [LocalStack Guide](doc/localstack.md) - Testing with LocalStack
- [Testing Guide](doc/testing.md) - Comprehensive testing documentation

## Contributing

This project follows Test-Driven Development (TDD) principles. When adding new features:

1. Write tests first
2. Implement the feature
3. Ensure all tests pass
4. Update documentation
