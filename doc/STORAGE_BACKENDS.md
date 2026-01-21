# Storage Backends

s3fs-go now supports multiple storage backends through a pluggable storage interface.

## Supported Backends

1. **S3** (default) - Amazon S3 and S3-compatible services
2. **PostgreSQL** - Store files in PostgreSQL database
3. **MongoDB** - Store files in MongoDB database

## Architecture

The filesystem uses a `storage.Backend` interface that abstracts storage operations:

```go
type Backend interface {
    Read(ctx context.Context, path string) ([]byte, error)
    ReadRange(ctx context.Context, path string, start, end int64) ([]byte, error)
    Write(ctx context.Context, path string, data []byte) error
    WriteWithMetadata(ctx context.Context, path string, data []byte, metadata map[string]string) error
    Delete(ctx context.Context, path string) error
    List(ctx context.Context, prefix string) ([]string, error)
    GetAttr(ctx context.Context, path string) (*Attr, error)
    Rename(ctx context.Context, oldPath, newPath string) error
    Exists(ctx context.Context, path string) (bool, error)
}
```

## Usage

### S3 Backend (Default)

```go
import (
    "github.com/s3fs-fuse/s3fs-go/internal/fuse"
    "github.com/s3fs-fuse/s3fs-go/internal/s3client"
    "github.com/s3fs-fuse/s3fs-go/internal/storage"
)

// Create S3 client
client := s3client.NewClient("bucket", "region", creds)

// Create filesystem (automatically uses S3 adapter)
fs := fuse.NewFilesystem(client)
```

### PostgreSQL Backend

```go
import (
    "github.com/s3fs-fuse/s3fs-go/internal/fuse"
    "github.com/s3fs-fuse/s3fs-go/internal/storage"
)

// Create PostgreSQL backend
backend, err := storage.NewBackend(storage.Config{
    Type:            storage.BackendTypePostgres,
    PostgresConnStr: "postgres://user:pass@localhost/dbname?sslmode=disable",
    PostgresTable:   "files",
    PostgresBucket:  "default",
})
if err != nil {
    log.Fatal(err)
}

// Create filesystem with backend
fs := fuse.NewFilesystemWithBackend(backend)
```

### MongoDB Backend

```go
import (
    "github.com/s3fs-fuse/s3fs-go/internal/fuse"
    "github.com/s3fs-fuse/s3fs-go/internal/storage"
)

// Create MongoDB backend
backend, err := storage.NewBackend(storage.Config{
    Type:            storage.BackendTypeMongoDB,
    MongoURI:        "mongodb://localhost:27017",
    MongoDatabase:   "s3fs",
    MongoCollection: "files",
    MongoBucket:     "default",
})
if err != nil {
    log.Fatal(err)
}

// Create filesystem with backend
fs := fuse.NewFilesystemWithBackend(backend)
```

## Implementation Status

### Minimal Features Implemented

All backends implement the core filesystem operations:
- ✅ Read file
- ✅ Write file
- ✅ Delete file
- ✅ List directory
- ✅ Get file attributes (size, mode, mtime, uid, gid)
- ✅ Rename file
- ✅ Check file existence

### Backend-Specific Details

#### PostgreSQL

- Stores files in a single table with BYTEA column for data
- Supports metadata storage in JSONB column
- Uses path as primary key
- Indexes on bucket and path prefix for efficient queries

#### MongoDB

- Stores files as documents with `_id` as path
- Supports metadata storage in document fields
- Uses MongoDB indexes for efficient queries
- Supports bucket namespacing

#### S3

- Uses existing S3 client implementation
- Maintains backward compatibility
- Supports all S3 features (multipart uploads, etc.)

## Future Enhancements

- [ ] Add Redis backend
- [ ] Add MySQL/MariaDB backend
- [ ] Add SQLite backend
- [ ] Add support for streaming large files
- [ ] Add transaction support for backends that support it
