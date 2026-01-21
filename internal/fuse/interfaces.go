package fuse

// This file documents all FUSE interfaces implemented by s3fs-go

/*
FUSE Interfaces Documentation

This file documents all FUSE interfaces from bazil.org/fuse/fs that are
implemented in s3fs-go. Each interface corresponds to specific filesystem
operations.

For more information, see: https://pkg.go.dev/bazil.org/fuse/fs
*/

// ============================================================================
// Filesystem-Level Interfaces
// ============================================================================

/*
FS Interface - Root filesystem node
Implemented by: FuseFS

type FS interface {
    Root() (Node, error)
}

Provides the root directory of the filesystem.
*/

// ============================================================================
// Node-Level Interfaces (Files and Directories)
// ============================================================================

/*
Node Interface - Base interface for all filesystem nodes
Implemented by: Dir, File

type Node interface {
    Attr(ctx context.Context, a *fuse.Attr) error
}

Provides file/directory attributes (size, mode, timestamps, ownership).
*/

/*
NodeStringLookuper Interface - Look up child nodes by name
Implemented by: Dir

type NodeStringLookuper interface {
    Lookup(ctx context.Context, name string) (Node, error)
}

Allows finding files and directories by name (e.g., "ls" command).
*/

/*
NodeMkdirer Interface - Create directories
Implemented by: Dir

type NodeMkdirer interface {
    Mkdir(ctx context.Context, req *fuse.MkdirRequest) (Node, error)
}

Creates a new directory (e.g., "mkdir" command).
*/

/*
NodeCreater Interface - Create files
Implemented by: Dir

type NodeCreater interface {
    Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (Node, Handle, error)
}

Creates a new file (e.g., "touch" or "> file").
*/

/*
NodeRemover Interface - Remove files and directories
Implemented by: Dir

type NodeRemover interface {
    Remove(ctx context.Context, req *fuse.RemoveRequest) error
}

Removes files or empty directories (e.g., "rm" and "rmdir" commands).
*/

/*
NodeSetattrer Interface - Set file attributes
Implemented by: Dir, File

type NodeSetattrer interface {
    Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error
}

Sets file attributes like permissions (chmod), ownership (chown), size (truncate).
*/

// ============================================================================
// Directory-Specific Interfaces
// ============================================================================

/*
HandleReadDirAller Interface - Read directory contents
Implemented by: Dir

type HandleReadDirAller interface {
    ReadDirAll(ctx context.Context) ([]fuse.Dirent, error)
}

Lists all entries in a directory (e.g., "ls" command).
*/

// ============================================================================
// File-Specific Interfaces
// ============================================================================

/*
NodeOpener Interface - Open files
Implemented by: File

type NodeOpener interface {
    Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (Handle, error)
}

Opens a file for reading/writing (e.g., when opening a file).
*/

/*
HandleReader Interface - Read file data
Implemented by: File

type HandleReader interface {
    Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error
}

Reads data from a file (supports range reads).
*/

/*
HandleWriter Interface - Write file data
Implemented by: File

type HandleWriter interface {
    Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error
}

Writes data to a file (supports offset writes).
*/

// ============================================================================
// Extended Attributes Interfaces
// ============================================================================

/*
NodeGetxattrer Interface - Get extended attribute
Implemented by: Dir, File

type NodeGetxattrer interface {
    Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error
}

Retrieves an extended attribute value.
*/

/*
NodeSetxattrer Interface - Set extended attribute
Implemented by: Dir, File

type NodeSetxattrer interface {
    Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error
}

Sets an extended attribute value.
*/

/*
NodeRemovexattrer Interface - Remove extended attribute
Implemented by: Dir, File

type NodeRemovexattrer interface {
    Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error
}

Removes an extended attribute.
*/

/*
NodeListxattrer Interface - List extended attributes
Implemented by: Dir, File

type NodeListxattrer interface {
    Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error
}

Lists all extended attribute names for a file/directory.
*/

// ============================================================================
// Currently Unimplemented Interfaces
// ============================================================================

/*
NodeSymlinker Interface - Create symbolic links
NOT IMPLEMENTED

type NodeSymlinker interface {
    Symlink(ctx context.Context, req *fuse.SymlinkRequest) (Node, error)
}

Would allow creating symbolic links (e.g., "ln -s").
*/

/*
NodeLinker Interface - Create hard links
NOT IMPLEMENTED

type NodeLinker interface {
    Link(ctx context.Context, req *fuse.LinkRequest, old Node) (Node, error)
}

Would allow creating hard links (e.g., "ln"). Note: Hard links are generally
not supported in object storage systems like S3.
*/

/*
NodeMknoder Interface - Create special files
NOT IMPLEMENTED

type NodeMknoder interface {
    Mknod(ctx context.Context, req *fuse.MknodRequest) (Node, error)
}

Would allow creating device files, named pipes, sockets. Note: Special files
are not supported in object storage systems like S3.
*/

/*
NodeAccesser Interface - Check file access permissions
NOT IMPLEMENTED

type NodeAccesser interface {
    Access(ctx context.Context, req *fuse.AccessRequest) error
}

Would allow checking if a file can be accessed with given permissions.
*/

/*
Statfser Interface - Filesystem statistics
NOT IMPLEMENTED

type Statfser interface {
    Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error
}

Would provide filesystem statistics (total space, free space, etc.) for
commands like "df".
*/

/*
HandleFlusher Interface - Flush file buffers
NOT IMPLEMENTED

type HandleFlusher interface {
    Flush(ctx context.Context, req *fuse.FlushRequest) error
}

Would ensure file data is written to storage before returning.
*/

/*
HandleFsyncer Interface - Sync file data
NOT IMPLEMENTED

type HandleFsyncer interface {
    Fsync(ctx context.Context, req *fuse.FsyncRequest) error
}

Would ensure file data is persisted to storage (fsync() syscall).
*/

/*
HandleReleaser Interface - Release file handle
NOT IMPLEMENTED

type HandleReleaser interface {
    Release(ctx context.Context, req *fuse.ReleaseRequest) error
}

Would clean up resources when a file is closed.
*/

/*
NodeOpendirer Interface - Open directory handle
NOT IMPLEMENTED

type NodeOpendirer interface {
    Opendir(ctx context.Context, req *fuse.OpendirRequest) (Handle, error)
}

Would open a directory handle for reading.
*/

// ============================================================================
// Implementation Summary
// ============================================================================

/*
Implemented Interfaces (15):

Filesystem Level:
  ✅ FS

Node Level:
  ✅ Node (base interface)
  ✅ NodeStringLookuper
  ✅ NodeMkdirer
  ✅ NodeCreater
  ✅ NodeRemover
  ✅ NodeSetattrer

Directory Operations:
  ✅ HandleReadDirAller

File Operations:
  ✅ NodeOpener
  ✅ HandleReader
  ✅ HandleWriter

Extended Attributes:
  ✅ NodeGetxattrer
  ✅ NodeSetxattrer
  ✅ NodeRemovexattrer
  ✅ NodeListxattrer

Not Implemented (9):
  ❌ NodeSymlinker
  ❌ NodeLinker
  ❌ NodeMknoder
  ❌ NodeAccesser
  ❌ Statfser
  ❌ HandleFlusher
  ❌ HandleFsyncer
  ❌ HandleReleaser
  ❌ NodeOpendirer

Total: 15 implemented, 9 not implemented
*/
