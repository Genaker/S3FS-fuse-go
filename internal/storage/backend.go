package storage

import (
	"context"
	"time"
)

// Attr represents file attributes
type Attr struct {
	Mode  uint32
	Size  int64
	Mtime time.Time
	Uid   uint32
	Gid   uint32
}

// Backend defines the interface for storage backends
// Minimal filesystem operations required for FUSE
type Backend interface {
	// Read reads file data
	Read(ctx context.Context, path string) ([]byte, error)
	
	// ReadRange reads a range of file data
	ReadRange(ctx context.Context, path string, start, end int64) ([]byte, error)
	
	// Write writes file data
	Write(ctx context.Context, path string, data []byte) error
	
	// WriteWithMetadata writes file data with metadata
	WriteWithMetadata(ctx context.Context, path string, data []byte, metadata map[string]string) error
	
	// Delete deletes a file
	Delete(ctx context.Context, path string) error
	
	// List lists objects with the given prefix (for directory listing)
	List(ctx context.Context, prefix string) ([]string, error)
	
	// GetAttr gets file attributes (size, mode, mtime, etc.)
	GetAttr(ctx context.Context, path string) (*Attr, error)
	
	// Rename renames a file or directory
	Rename(ctx context.Context, oldPath, newPath string) error
	
	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)
}
