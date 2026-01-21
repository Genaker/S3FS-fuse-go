# Missing FUSE Features

This document lists FUSE operations and features that are currently missing from the s3fs-go implementation.

**Last Updated:** January 2025  
**Current FUSE Package Coverage:** 49.4%

## Recent Implementations

The following features have been recently implemented with comprehensive test coverage:

- ✅ **Mkdir** (90.9% coverage) - Directory creation with `.keep` markers
- ✅ **Rmdir** (66.7% coverage) - Empty directory removal
- ✅ **Rename** (78.9% coverage) - File rename/move with multipart support for large files
- ✅ **Utimens** (85.7% coverage) - Set file access and modification times
- ✅ **Extended Attributes** (82-91% coverage) - Full xattr support for files and directories
- ✅ **WriteFile** (55.3% coverage) - Advanced write operations (append, overwrite, truncate, middle writes)

All new features include comprehensive integration tests in `internal/integration/fuse/` using LocalStack.

## Currently Implemented Features

✅ **Basic File Operations:**
- `Attr` - Get file/directory attributes
- `Lookup` - Find files/directories
- `ReadDirAll` - List directory contents
- `Open` - Open files
- `Read` - Read file data (with range support)
- `Write` - Write file data (with offset support)
- `Create` - Create new files
- `Remove` - Delete files
- `Rename` - Rename/move files

✅ **Directory Operations:**
- `Mkdir` - Create directories (NEW)
- `Rmdir` - Remove empty directories (NEW)

✅ **File Attributes:**
- `Setattr` - Set file attributes (chmod, chown)
- `Chmod` - Change file permissions
- `Chown` - Change file ownership
- `Utimens` - Set file times (fully implemented and tested)

✅ **Extended Attributes:**
- `Getxattr` - Get extended attribute (fully implemented and tested)
- `Setxattr` - Set extended attribute (fully implemented and tested)
- `Removexattr` - Remove extended attribute (fully implemented and tested)
- `Listxattr` - List extended attributes (fully implemented and tested)
- Support for extended attributes on both files and directories

✅ **Advanced Features:**
- Multi-part upload support (for files > 5MB)
- Multi-part copy support (for large file renames)
- Range reads (partial file reads)
- Metadata handling (file times, permissions, extended attributes)
- Write operations with offset support (append, overwrite, truncate)

## Missing FUSE Operations

### 🔴 Critical Missing Operations

#### 1. **Symlink** - Create Symbolic Link
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.NodeSymlinker`  
**Description:** Create symbolic links  
**Impact:** `ln -s` command won't work, symlinks not supported

```go
// Missing interface:
type NodeSymlinker interface {
    Symlink(ctx context.Context, req *fuse.SymlinkRequest) (Node, error)
}
```

#### 2. **Link** - Create Hard Link
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.NodeLinker`  
**Description:** Create hard links  
**Note:** Hard links are generally not supported in object storage  
**Impact:** `ln` command won't work

```go
// Missing interface:
type NodeLinker interface {
    Link(ctx context.Context, req *fuse.LinkRequest, old Node) (Node, error)
}
```

#### 3. **Mknod** - Create Special File
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.NodeMknoder`  
**Description:** Create device files, named pipes, sockets  
**Impact:** `mknod` command won't work

```go
// Missing interface:
type NodeMknoder interface {
    Mknod(ctx context.Context, req *fuse.MknodRequest) (Node, error)
}
```

### 🟡 Important Missing Operations

#### 3. **Rename** - Rename/Move Files
**Status:** ✅ **IMPLEMENTED**  
**FUSE Interface:** `fs.NodeRenamer`  
**Description:** Rename or move files  
**Implementation:** Supports both small files (simple copy) and large files (multipart copy for >5MB)  
**Coverage:** 78.9%  
**Tests:**
- Integration tests: `TestRename`, `TestRenameLargeFile`, `TestRenameNonExistent` (in `internal/integration/fuse/`)

#### 4. **Utimens** - Set File Times
**Status:** ✅ **IMPLEMENTED**  
**FUSE Interface:** Via `Setattr`  
**Description:** Set access and modification times  
**Implementation:** Updates metadata for both files and directories  
**Coverage:** 85.7%  
**Tests:**
- Integration tests: `TestUtimens`, `TestUtimensDirectory` (in `internal/integration/fuse/`)

#### 5. **Extended Attributes (xattr)** - Full Support
**Status:** ✅ **IMPLEMENTED**  
**FUSE Interfaces:** `fs.NodeGetxattrer`, `fs.NodeSetxattrer`, `fs.NodeRemovexattrer`, `fs.NodeListxattrer`  
**Description:** Full extended attribute support for files and directories  
**Coverage:**
- `SetXattr`: 85.7%
- `GetXattr`: 90.5%
- `ListXattr`: 91.3%
- `RemoveXattr`: 82.8%
**Tests:**
- Integration tests: `TestSetXattr`, `TestGetXattr`, `TestGetXattrNonExistent`, `TestListXattr`, `TestRemoveXattr`, `TestXattrDirectory` (in `internal/integration/fuse/`)

#### 6. **Access** - Check File Permissions
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.NodeAccesser`  
**Description:** Check if file can be accessed with given permissions  
**Impact:** Some programs use `access()` syscall to check permissions

```go
// Missing interface:
type NodeAccesser interface {
    Access(ctx context.Context, req *fuse.AccessRequest) error
}
```

#### 7. **WriteFile** - Advanced Write Operations
**Status:** ✅ **IMPLEMENTED**  
**Description:** Write operations with offset support (append, overwrite, truncate, middle writes)  
**Coverage:** 55.3%  
**Tests:**
- Integration tests: `TestWriteFileAppend`, `TestWriteFileOverwrite`, `TestWriteFileTruncate`, `TestWriteFileLarge`, `TestWriteFileMiddle` (in `internal/integration/fuse/`)

#### 8. **Statfs** - Filesystem Statistics
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.Statfser`  
**Description:** Return filesystem statistics (total space, free space, etc.)  
**Impact:** `df` command won't show correct information

```go
// Missing interface:
type Statfser interface {
    Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error
}
```

#### 9. **Flush** - Flush File Buffers
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.HandleFlusher`  
**Description:** Flush cached file data to storage  
**Impact:** Data may not be persisted immediately after write

```go
// Missing interface:
type HandleFlusher interface {
    Flush(ctx context.Context, req *fuse.FlushRequest) error
}
```

#### 10. **Fsync** - Sync File Data
**Status:** ❌ Not implemented  
**FUSE Interface:** `fs.HandleFsyncer`  
**Description:** Ensure file data is written to storage  
**Impact:** `fsync()` calls won't work, data integrity concerns

```go
// Missing interface:
type HandleFsyncer interface {
    Fsync(ctx context.Context, req *fuse.FsyncRequest) error
}
```

#### 11. **Release** - Close File Handle
**Status:** ⚠️ Partially implemented  
**FUSE Interface:** `fs.HandleReleaser`  
**Description:** Clean up when file is closed  
**Current:** Basic cleanup exists, but no explicit release handler  
**Impact:** File handles may not be properly cleaned up

```go
// Missing interface:
type HandleReleaser interface {
    Release(ctx context.Context, req *fuse.ReleaseRequest) error
}
```

#### 12. **Opendir** - Open Directory Handle
**Status:** ⚠️ Partially implemented  
**FUSE Interface:** `fs.NodeOpendirer`  
**Description:** Open directory for reading  
**Current:** Directories work but no explicit opendir handler  
**Impact:** Some directory operations may not work optimally

```go
// Missing interface:
type NodeOpendirer interface {
    Opendir(ctx context.Context, req *fuse.OpendirRequest) (Handle, error)
}
```

## Missing Advanced Features

### Caching

#### 12. **File Descriptor Cache (FdCache)**
**Status:** ❌ Not implemented  
**Description:** Cache file data locally for faster access  
**Impact:** Every read requires network request, slower performance

#### 13. **Metadata Cache (Stat Cache)**
**Status:** ❌ Not implemented  
**Description:** Cache file attributes to reduce HEAD requests  
**Impact:** Many HEAD requests to S3, slower directory listings

#### 14. **Cache Management**
**Status:** ❌ Not implemented  
**Description:** Cache size limits, expiration, cleanup  
**Impact:** No control over cache behavior

### Performance Optimizations

#### 15. **Write Buffering**
**Status:** ⚠️ Partial  
**Description:** Buffer writes before uploading  
**Current:** Writes are immediate  
**Impact:** Many small writes create many S3 requests

#### 16. **Read-Ahead Caching**
**Status:** ❌ Not implemented  
**Description:** Prefetch data when reading sequentially  
**Impact:** Sequential reads are slower than necessary

#### 17. **Connection Pooling**
**Status:** ⚠️ Partial (via AWS SDK)  
**Description:** Reuse HTTP connections  
**Current:** AWS SDK handles this, but no explicit control

### Consistency & Concurrency

#### 18. **File Locking**
**Status:** ❌ Not implemented  
**Description:** POSIX file locking (flock, fcntl)  
**Impact:** Multiple processes can't coordinate file access

#### 19. **Change Notifications**
**Status:** ❌ Not implemented  
**Description:** inotify support for file changes  
**Impact:** Programs using inotify won't detect changes

#### 20. **External Change Detection**
**Status:** ❌ Not implemented  
**Description:** Detect changes made outside the mount  
**Impact:** Changes made via AWS console won't be visible

### Mount Options

#### 21. **Mount Option Support**
**Status:** ⚠️ Partial  
**Description:** Support for FUSE mount options  
**Current:** Basic mount options only  
**Missing:**
- `allow_other` - Allow other users to access
- `default_permissions` - Use kernel permission checking
- `atime`, `noatime` - Access time updates
- `sync`, `async` - Synchronous/asynchronous I/O
- `ro`, `rw` - Read-only/read-write mode

### Error Handling

#### 22. **Better Error Codes**
**Status:** ⚠️ Partial  
**Description:** Return proper POSIX error codes  
**Current:** Basic error handling exists  
**Impact:** Some errors may not be properly reported

## Implementation Priority

### High Priority (Core Functionality)
1. ✅ **Mkdir** - Explicit directory creation (IMPLEMENTED - 90.9% coverage)
2. ✅ **Rmdir** - Directory removal (IMPLEMENTED - 66.7% coverage)
3. ✅ **Rename** - File rename/move (IMPLEMENTED - 78.9% coverage)
4. ✅ **Utimens** - Set file times (IMPLEMENTED - 85.7% coverage)
5. ✅ **Extended Attributes** - Full xattr support (IMPLEMENTED - 82-91% coverage)
6. ✅ **WriteFile** - Advanced write operations (IMPLEMENTED - 55.3% coverage)
7. **Flush** - Ensure data persistence
8. **Fsync** - Data integrity
9. **Statfs** - Filesystem statistics

### Medium Priority (Common Operations)
10. **Symlink** - Symbolic link support
11. **Access** - Permission checking
12. **Release** - Proper cleanup
13. **Opendir** - Directory handles

### Low Priority (Advanced Features)
14. **Link** - Hard links (may not be feasible with S3)
15. **Mknod** - Special files (may not be feasible with S3)
16. **File locking** - Coordination between processes
17. **inotify** - Change notifications

### Performance (Important but not blocking)
14. **Metadata cache** - Reduce HEAD requests
15. **File cache** - Faster reads
16. **Write buffering** - Reduce PUT requests

## Current Test Coverage

**Overall FUSE Package Coverage:** 49.4%

**Coverage by Operation:**
- `Mkdir`: 90.9%
- `Rmdir`: 66.7%
- `Rename`: 78.9%
- `Utimens`: 85.7%
- `WriteFile`: 55.3%
- `SetXattr`: 85.7%
- `GetXattr`: 90.5%
- `ListXattr`: 91.3%
- `RemoveXattr`: 82.8%

**Test Organization:**
- Unit tests: Located in `internal/fuse/*_test.go`
- Integration tests: Located in `internal/integration/fuse/` (requires `-tags=integration`)
- LocalStack integration tests: Default provider for integration tests

## Notes

- **Hard Links & Mknod:** These may not be feasible with S3's object storage model, as S3 doesn't support multiple names for the same object or special files.

- **Symlinks:** Can be implemented by storing symlink target in S3 metadata or as a special object.

- **Caching:** Critical for performance but adds complexity. Should be implemented after core operations are complete.

- **Mount Options:** Many can be handled by FUSE itself, but some require explicit implementation.

- **Integration Tests:** All integration tests are now organized in `internal/integration/` folder and use LocalStack by default. Tests can be run with production S3 or R2 by setting `S3_PROVIDER` environment variable.

- **Test Coverage:** Coverage has improved significantly with comprehensive integration tests. Focus areas for improvement: WriteFile edge cases, FUSE wrapper functions (tested indirectly).

## References

- [FUSE Operations](https://libfuse.github.io/doxygen/structfuse__operations.html)
- [bazil.org/fuse Documentation](https://pkg.go.dev/bazil.org/fuse/fs)
