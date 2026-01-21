package fuse

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
)

// Attr represents file attributes
type Attr struct {
	Mode  os.FileMode
	Size  int64
	Mtime time.Time
	Uid   uint32
	Gid   uint32
}

// DirEntry represents a directory entry
type DirEntry struct {
	Name  string
	IsDir bool
}

// Filesystem represents the S3 FUSE filesystem
type Filesystem struct {
	client *s3client.Client
}

// NewFilesystem creates a new filesystem instance
func NewFilesystem(client *s3client.Client) *Filesystem {
	return &Filesystem{
		client: client,
	}
}

// normalizePath normalizes S3 path (removes leading slash, ensures trailing slash for directories)
func (fs *Filesystem) normalizePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	return path
}

// GetAttr retrieves file attributes
func (fs *Filesystem) GetAttr(ctx context.Context, path string) (*Attr, error) {
	normalizedPath := fs.normalizePath(path)
	
	// Check if it's a directory by listing
	if normalizedPath == "" || strings.HasSuffix(normalizedPath, "/") {
		return &Attr{
			Mode:  os.ModeDir | 0755,
			Size:  4096,
			Mtime: time.Now(),
			Uid:   uint32(os.Getuid()),
			Gid:   uint32(os.Getgid()),
		}, nil
	}

	// Try to get object metadata
	metadata, err := fs.client.HeadObject(ctx, normalizedPath)
	if err != nil {
		// Check if it's a directory by listing objects with this prefix
		objects, listErr := fs.client.ListObjects(ctx, normalizedPath+"/")
		if listErr == nil && len(objects) > 0 {
			return &Attr{
				Mode:  os.ModeDir | 0755,
				Size:  4096,
				Mtime: time.Now(),
				Uid:   uint32(os.Getuid()),
				Gid:   uint32(os.Getgid()),
			}, nil
		}
		return nil, fmt.Errorf("file not found: %w", syscall.ENOENT)
	}

	// Get object size from metadata (more efficient than downloading)
	size, err := fs.client.HeadObjectSize(ctx, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get object size: %w", err)
	}

	// Parse metadata for mode, uid, gid, mtime
	mode := os.FileMode(0644)
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	mtime := time.Now()

	if modeStr, ok := metadata["x-amz-meta-mode"]; ok {
		var modeVal uint32
		fmt.Sscanf(modeStr, "%o", &modeVal)
		mode = os.FileMode(modeVal)
	}
	if uidStr, ok := metadata["x-amz-meta-uid"]; ok {
		fmt.Sscanf(uidStr, "%d", &uid)
	}
	if gidStr, ok := metadata["x-amz-meta-gid"]; ok {
		fmt.Sscanf(gidStr, "%d", &gid)
	}
	if mtimeStr, ok := metadata["x-amz-meta-mtime"]; ok {
		var unixTime int64
		if _, err := fmt.Sscanf(mtimeStr, "%d", &unixTime); err == nil {
			mtime = time.Unix(unixTime, 0)
		} else if parsed, err := time.Parse(time.RFC3339, mtimeStr); err == nil {
			mtime = parsed
		}
	}

	return &Attr{
		Mode:  mode,
		Size:  size,
		Mtime: mtime,
		Uid:   uid,
		Gid:   gid,
	}, nil
}

// ReadDir lists directory entries
func (fs *Filesystem) ReadDir(ctx context.Context, path string) ([]DirEntry, error) {
	normalizedPath := fs.normalizePath(path)
	if normalizedPath != "" && !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	objects, err := fs.client.ListObjects(ctx, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Track seen directory names to avoid duplicates
	seen := make(map[string]bool)
	entries := make([]DirEntry, 0)

	for _, objKey := range objects {
		// Remove the prefix to get relative path
		relativePath := strings.TrimPrefix(objKey, normalizedPath)
		if relativePath == "" {
			continue
		}

		// Extract first component (file or directory name)
		parts := strings.Split(relativePath, "/")
		name := parts[0]

		if seen[name] {
			continue
		}
		seen[name] = true

		isDir := len(parts) > 1
		entries = append(entries, DirEntry{
			Name:  name,
			IsDir: isDir,
		})
	}

	return entries, nil
}

// ReadFile reads file data
func (fs *Filesystem) ReadFile(ctx context.Context, path string, offset int64, size int64) ([]byte, error) {
	normalizedPath := fs.normalizePath(path)
	
	// Use range read if offset or size is specified
	var end int64
	if size > 0 {
		end = offset + size - 1
	}
	
	data, err := fs.client.GetObjectRange(ctx, normalizedPath, offset, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return data, nil
}

// WriteFile writes file data
func (fs *Filesystem) WriteFile(ctx context.Context, path string, data []byte, offset int64) error {
	normalizedPath := fs.normalizePath(path)
	
	// Simple write (full file replacement)
	if offset == 0 {
		// Use multipart for large files
		if int64(len(data)) >= 5*1024*1024 {
			return fs.client.PutObjectMultipart(ctx, normalizedPath, data)
		}
		return fs.client.PutObject(ctx, normalizedPath, data)
	}

	// For non-zero offset, we need to read existing file, modify, and write back
	existing, err := fs.client.GetObject(ctx, normalizedPath)
	if err != nil {
		// File doesn't exist, create new
		if offset > 0 {
			// Pad with zeros up to offset
			padded := make([]byte, offset)
			data = append(padded, data...)
		}
		// Use multipart for large files
		if int64(len(data)) >= 5*1024*1024 {
			return fs.client.PutObjectMultipart(ctx, normalizedPath, data)
		}
		return fs.client.PutObject(ctx, normalizedPath, data)
	}

	// Modify existing file
	if offset >= int64(len(existing)) {
		// Extend file
		padded := make([]byte, offset-int64(len(existing)))
		existing = append(existing, padded...)
		existing = append(existing, data...)
	} else {
		// Overwrite part of file
		before := existing[:offset]
		afterOffset := offset + int64(len(data))
		var after []byte
		if afterOffset < int64(len(existing)) {
			after = existing[afterOffset:]
		}
		existing = append(before, append(data, after...)...)
	}

	// Update mtime/ctime when writing
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-mtime": fmt.Sprintf("%d", now.Unix()),
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
	}

	// Use multipart for large files
	if int64(len(existing)) >= 5*1024*1024 {
		// For multipart, we need to preserve existing metadata
		existingMeta, _ := fs.client.HeadObject(ctx, normalizedPath)
		if existingMeta != nil {
			for k, v := range existingMeta {
				if !strings.HasPrefix(k, "x-amz-meta-mtime") && !strings.HasPrefix(k, "x-amz-meta-ctime") {
					metadata[k] = v
				}
			}
		}
		// Preserve existing metadata
		existingMeta, err := fs.client.HeadObject(ctx, normalizedPath)
		if err == nil && existingMeta != nil {
			for k, v := range existingMeta {
				if !strings.HasPrefix(k, "x-amz-meta-mtime") && !strings.HasPrefix(k, "x-amz-meta-ctime") {
					metadata[k] = v
				}
			}
		}
		// For multipart, metadata is set during CreateMultipartUpload
		// For now, just upload without metadata preservation in multipart
		return fs.client.PutObjectMultipart(ctx, normalizedPath, existing)
	}
	return fs.client.PutObjectWithMetadata(ctx, normalizedPath, existing, metadata)
}

// Create creates a new file
func (fs *Filesystem) Create(ctx context.Context, path string, mode os.FileMode) error {
	normalizedPath := fs.normalizePath(path)
	// Create empty file
	return fs.client.PutObject(ctx, normalizedPath, []byte{})
}

// Remove removes a file
func (fs *Filesystem) Remove(ctx context.Context, path string) error {
	normalizedPath := fs.normalizePath(path)
	return fs.client.DeleteObject(ctx, normalizedPath)
}

// Rename renames a file
func (fs *Filesystem) Rename(ctx context.Context, oldPath, newPath string) error {
	oldNormalized := fs.normalizePath(oldPath)
	newNormalized := fs.normalizePath(newPath)

	// Get source file size to determine if we should use multipart copy
	sourceSize, err := fs.client.HeadObjectSize(ctx, oldNormalized)
	if err != nil {
		return fmt.Errorf("failed to get source file size: %w", err)
	}

	// Use multipart copy for large files, otherwise use simple copy
	if sourceSize >= 5*1024*1024 {
		err = fs.client.CopyObjectMultipart(ctx, oldNormalized, newNormalized)
		if err != nil {
			return fmt.Errorf("failed to copy large file: %w", err)
		}
	} else {
		// Read old file
		data, err := fs.client.GetObject(ctx, oldNormalized)
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}

		// Write to new location
		err = fs.client.PutObject(ctx, newNormalized, data)
		if err != nil {
			return fmt.Errorf("failed to write destination file: %w", err)
		}
	}

	// Delete old file
	err = fs.client.DeleteObject(ctx, oldNormalized)
	if err != nil {
		return fmt.Errorf("failed to delete source file: %w", err)
	}

	return nil
}

// Mkdir creates a directory
func (fs *Filesystem) Mkdir(ctx context.Context, path string, mode os.FileMode) error {
	normalizedPath := fs.normalizePath(path)
	
	// Ensure path ends with / for directories
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}
	
	// Check if directory already exists
	entries, err := fs.ReadDir(ctx, path)
	if err == nil && len(entries) >= 0 {
		// Directory might exist, check explicitly
		attr, err := fs.GetAttr(ctx, path)
		if err == nil && attr.Mode.IsDir() {
			return syscall.EEXIST // Directory already exists
		}
	}
	
	// Create directory marker object (empty object with trailing slash)
	// Store metadata for mode, uid, gid
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-mode":  fmt.Sprintf("%o", mode),
		"x-amz-meta-uid":   fmt.Sprintf("%d", os.Getuid()),
		"x-amz-meta-gid":   fmt.Sprintf("%d", os.Getgid()),
		"x-amz-meta-mtime": fmt.Sprintf("%d", now.Unix()),
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
	}
	
	// Create directory marker (empty object)
	return fs.client.PutObjectWithMetadata(ctx, normalizedPath+".keep", []byte{}, metadata)
}

// Rmdir removes an empty directory
func (fs *Filesystem) Rmdir(ctx context.Context, path string) error {
	normalizedPath := fs.normalizePath(path)
	
	// Ensure path ends with / for directories
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}
	
	// Check if directory exists
	attr, err := fs.GetAttr(ctx, path)
	if err != nil {
		return syscall.ENOENT
	}
	if !attr.Mode.IsDir() {
		return syscall.ENOTDIR
	}
	
	// Check if directory is empty
	entries, err := fs.ReadDir(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to list directory: %w", err)
	}
	
	// Filter out directory markers
	realEntries := 0
	for _, entry := range entries {
		if entry.Name != ".keep" {
			realEntries++
		}
	}
	
	if realEntries > 0 {
		return syscall.ENOTEMPTY // Directory is not empty
	}
	
	// Remove directory marker if it exists
	err = fs.client.DeleteObject(ctx, normalizedPath+".keep")
	if err != nil {
		// Directory marker might not exist, which is okay
		// Check if there are any objects with this prefix
		objects, listErr := fs.client.ListObjects(ctx, normalizedPath)
		if listErr != nil || len(objects) > 0 {
			return syscall.ENOTEMPTY
		}
		// Directory is effectively empty, allow removal
		return nil
	}
	
	return nil
}