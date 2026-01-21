# Test Organization

This document describes the test structure for s3fs-go following Go best practices.

## Test Structure

### Standard (Unit Tests)
**Location:** `internal/{package}/{package}_test.go`  
**Example:** `internal/cache/stat_cache_test.go`  
**Purpose:** Unit testing logic and private helpers  
**Build Tag:** None (runs by default)

These tests:
- Test individual functions and methods in isolation
- Use mocks or minimal dependencies
- Run fast without external services
- Examples:
  - `internal/cache/stat_cache_test.go` - Tests stat cache logic
  - `internal/cache/fd_cache_test.go` - Tests FD cache logic
  - `internal/fuse/permissions_test.go` - Tests chmod/chown logic
  - `internal/fuse/xattr_test.go` - Tests extended attributes logic
  - `internal/s3client/client_test.go` - Tests S3 client unit logic

### Integration Tests
**Location:** `tests/integration_*.go`  
**Example:** `tests/integration_fuse_comprehensive_test.go`  
**Purpose:** Testing how multiple packages work together or DB connections  
**Build Tag:** `//go:build integration`

These tests:
- Test interactions between multiple packages
- Require external services (LocalStack, S3, R2)
- Test end-to-end workflows
- Examples:
  - `tests/integration_fuse_comprehensive_test.go` - Comprehensive FUSE operations
  - `tests/integration_fuse_filesystem_test.go` - Filesystem operations
  - `tests/integration_fuse_missing_ops_test.go` - Missing FUSE operations
  - `tests/integration_s3client_test.go` - S3 client integration
  - `tests/testhelper.go` - Shared test helpers

**Running Integration Tests:**
```bash
# With LocalStack
go test -tags=integration ./tests/... -v

# With specific provider
S3_PROVIDER=localstack go test -tags=integration ./tests/... -v
S3_PROVIDER=s3 go test -tags=integration ./tests/... -v
S3_PROVIDER=r2 go test -tags=integration ./tests/... -v
```

### Functional Tests
**Location:** `cmd/{app}/main_test.go`  
**Example:** `cmd/s3fs/main_test.go`  
**Purpose:** End-to-end testing of the CLI or Entry point  
**Build Tag:** `//go:build functional`

These tests:
- Test the CLI application as a whole
- Test command-line arguments and flags
- Test application lifecycle
- Examples:
  - `cmd/s3fs/main_test.go` - CLI argument validation, help, error handling

**Running Functional Tests:**
```bash
go test -tags=functional ./cmd/... -v
```

## Test Organization Summary

| Type | Location | Build Tag | Purpose |
|------|----------|-----------|---------|
| **Standard** | `internal/{pkg}/{pkg}_test.go` | None | Unit testing logic |
| **Integration** | `tests/integration_*.go` | `integration` | Multi-package testing |
| **Functional** | `cmd/{app}/main_test.go` | `functional` | CLI/entry point testing |

## Running Tests

### All Unit Tests (Standard)
```bash
go test ./internal/... -v
```

### All Integration Tests
```bash
# Ensure LocalStack is running
docker-compose -f docker-compose.localstack.yml up -d

# Run integration tests
go test -tags=integration ./tests/... -v
```

### All Functional Tests
```bash
go test -tags=functional ./cmd/... -v
```

### All Tests
```bash
# Unit tests
go test ./internal/... -v

# Integration tests
go test -tags=integration ./tests/... -v

# Functional tests
go test -tags=functional ./cmd/... -v
```

## Test Helpers

### `tests/testhelper.go`
Provides shared utilities for integration tests:
- `SetupTestClient()` - Creates S3 client based on provider
- `SetupTestFilesystem()` - Creates filesystem for testing
- `RequireLocalStack()` - Ensures LocalStack is available
- `GetProvider()` - Gets S3 provider from environment

## Migration Notes

Tests were reorganized from:
- `internal/integration/` → `tests/`
- Integration tests in `internal/fuse/` → `tests/`
- Integration tests in `internal/s3client/` → `tests/`

Unit tests remain in their original locations alongside source files.
