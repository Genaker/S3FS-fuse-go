# Testing Guide

This document describes how to run tests for s3fs-go.

## Test Types

### Unit Tests

Unit tests test individual components in isolation without external dependencies.

```bash
# Run all unit tests
go test ./...

# Run specific package unit tests
go test ./internal/credentials
go test ./internal/s3client
go test ./internal/fuse
```

### Integration Tests

Integration tests require S3 credentials and will be skipped if credentials are not available.

```bash
# Run all tests (integration tests will skip without credentials)
go test ./... -v

# Run integration tests with credentials
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
go test ./... -v
```

### LocalStack Integration Tests

LocalStack tests run against a local S3-compatible service. See [LocalStack Documentation](localstack.md) for details.

```bash
# Start LocalStack first
docker-compose -f docker-compose.localstack.yml up -d

# Run LocalStack tests
go test -v ./internal/s3client -run TestLocalStack
```

## Running Tests

### Local Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific test
go test -v ./internal/fuse -run TestCreateEmptyFile

# Run tests with coverage
go test ./... -cover
```

### Fast Docker Testing (Recommended)

For faster test execution, use the persistent Docker container:

```bash
# First time: creates container
./run_tests_fast.sh

# Subsequent runs: reuses container
./run_tests_fast.sh

# Run specific tests
docker exec s3fs-go-test go test -v ./internal/s3client -run TestLocalStack
```

### Standard Docker Testing

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

## Test Coverage

### View Coverage Report

```bash
# Generate coverage profile
go test ./... -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Coverage by Package

```bash
# Coverage for specific package
go test ./internal/s3client -cover
go test ./internal/fuse -cover
go test ./internal/credentials -cover
```

## Test Structure

```
go/
├── internal/
│   ├── credentials/
│   │   ├── credentials.go
│   │   └── credentials_test.go      # Unit tests
│   ├── s3client/
│   │   ├── client.go
│   │   ├── client_test.go           # Unit tests
│   │   ├── integration_test.go      # Integration tests (require credentials)
│   │   └── localstack_integration_test.go  # LocalStack tests
│   └── fuse/
│       ├── filesystem.go
│       ├── filesystem_test.go        # Unit tests
│       └── integration_test.go      # Integration tests (require credentials)
```

## Writing Tests

### Unit Test Example

```go
func TestNewClient(t *testing.T) {
    client := s3client.NewClient("test-bucket", "us-east-1", nil)
    if client == nil {
        t.Fatal("NewClient returned nil")
    }
}
```

### Integration Test Example

```go
func TestPutGetObject(t *testing.T) {
    client := setupTestClient(t) // Skip if no credentials
    ctx := context.Background()
    
    err := client.PutObject(ctx, "test-key", []byte("test"))
    if err != nil {
        t.Fatalf("PutObject failed: %v", err)
    }
    
    data, err := client.GetObject(ctx, "test-key")
    if err != nil {
        t.Fatalf("GetObject failed: %v", err)
    }
    
    if string(data) != "test" {
        t.Errorf("Expected 'test', got '%s'", string(data))
    }
}
```

### LocalStack Test Example

```go
func TestLocalStackPutGet(t *testing.T) {
    client := setupLocalStackTest(t) // Skip if LocalStack not running
    ctx := context.Background()
    
    // Test implementation
}
```

## Test Best Practices

1. **Use Table-Driven Tests**: For testing multiple scenarios
2. **Clean Up**: Always clean up test data
3. **Skip Appropriately**: Skip tests when dependencies aren't available
4. **Use Context**: Always use context.Context for operations
5. **Test Edge Cases**: Test error conditions and edge cases
6. **Isolate Tests**: Tests should not depend on each other

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test ./... -v
      - name: Run LocalStack tests
        run: |
          docker-compose -f docker-compose.localstack.yml up -d
          sleep 10
          go test -v ./internal/s3client -run TestLocalStack
```

## Troubleshooting

### Tests Skipping Unexpectedly

- Check if credentials are set (for integration tests)
- Verify LocalStack is running (for LocalStack tests)
- Check test output for skip reasons

### Slow Test Execution

- Use the persistent Docker container (`run_tests_fast.sh`)
- Run tests in parallel: `go test ./... -parallel 4`
- Run only relevant tests instead of all tests

### Test Failures

- Check error messages for details
- Verify dependencies are installed
- Ensure test environment is set up correctly
- Check network connectivity for integration tests
