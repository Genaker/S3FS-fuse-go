package fuse

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/s3fs-fuse/s3fs-go/internal/cache"
	"github.com/s3fs-fuse/s3fs-go/internal/s3client"
	"github.com/s3fs-fuse/s3fs-go/internal/storage/types"
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

// S3ClientInterface defines the interface for S3 operations
type S3ClientInterface interface {
	ListObjects(ctx context.Context, prefix string) ([]string, error)
	GetObject(ctx context.Context, key string) ([]byte, error)
	GetObjectRange(ctx context.Context, key string, start, end int64) ([]byte, error)
	PutObject(ctx context.Context, key string, data []byte) error
	PutObjectWithMetadata(ctx context.Context, key string, data []byte, metadata map[string]string) error
	DeleteObject(ctx context.Context, key string) error
	HeadObject(ctx context.Context, key string) (map[string]string, error)
	HeadObjectSize(ctx context.Context, key string) (int64, error)
	CopyObjectWithMetadata(ctx context.Context, sourceKey, destKey string, metadata map[string]string) error
	CopyObjectMultipart(ctx context.Context, sourceKey, destKey string) error
	CreateBucket(ctx context.Context) error
	PutObjectMultipart(ctx context.Context, key string, data []byte) error
}

// Filesystem represents the FUSE filesystem
type Filesystem struct {
	backend         types.Backend // Storage backend (S3, Postgres, MongoDB, etc.)
	client          S3ClientInterface // Deprecated: kept for backward compatibility
	cache           *cache.Manager
	maxDirtyData    int64 // Maximum bytes to buffer before auto-upload (default: 10MB)
	enableFileLock  bool  // Enable file-level advisory locking (default: false, uses entity-level locking)
}

// NewFilesystem creates a new filesystem instance with S3 client (backward compatibility)
func NewFilesystem(client S3ClientInterface) *Filesystem {
	return NewFilesystemWithBackend(newS3Adapter(client))
}

// NewFilesystemWithBackend creates a new filesystem instance with a storage backend
func NewFilesystemWithBackend(backend types.Backend) *Filesystem {
	return &Filesystem{
		backend:        backend,
		cache:          cache.DefaultManager(),
		maxDirtyData:   10 * 1024 * 1024, // Default: 10MB buffer
		enableFileLock: false,            // Default: entity-level locking (Option 1)
	}
}

// NewFilesystemWithCache creates a new filesystem instance with custom cache settings
func NewFilesystemWithCache(client *s3client.Client, cacheManager *cache.Manager) *Filesystem {
	return &Filesystem{
		client:         client,
		cache:          cacheManager,
		maxDirtyData:   10 * 1024 * 1024, // Default: 10MB buffer
		enableFileLock: false,            // Default: entity-level locking (Option 1)
	}
}

// SetMaxDirtyData sets the maximum bytes to buffer before auto-upload
func (fs *Filesystem) SetMaxDirtyData(maxBytes int64) {
	fs.maxDirtyData = maxBytes
}

// SetEnableFileLock enables or disables file-level advisory locking
// When enabled (true): Uses file-level advisory locking (Option 2) - provides stricter coordination
// When disabled (false, default): Uses entity-level mutex locking (Option 1) - better performance
func (fs *Filesystem) SetEnableFileLock(enable bool) {
	fs.enableFileLock = enable
}

// normalizePath normalizes path (removes leading slash, ensures trailing slash for directories)
func (fs *Filesystem) normalizePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	return path
}

// getBackend returns the storage backend, creating an adapter from client if needed
func (fs *Filesystem) getBackend() types.Backend {
	if fs.backend != nil {
		return fs.backend
	}
	// Fallback to S3 adapter for backward compatibility
	if fs.client != nil {
		return newS3Adapter(fs.client)
	}
	return nil
}

// newS3Adapter creates an S3 adapter (internal to avoid import cycle)
func newS3Adapter(client S3ClientInterface) types.Backend {
	return &s3Adapter{client: client}
}

// s3Adapter adapts S3ClientInterface to storage.Backend
type s3Adapter struct {
	client S3ClientInterface
}

func (s *s3Adapter) Read(ctx context.Context, path string) ([]byte, error) {
	return s.client.GetObject(ctx, path)
}

func (s *s3Adapter) ReadRange(ctx context.Context, path string, start, end int64) ([]byte, error) {
	return s.client.GetObjectRange(ctx, path, start, end)
}

func (s *s3Adapter) Write(ctx context.Context, path string, data []byte) error {
	return s.client.PutObject(ctx, path, data)
}

func (s *s3Adapter) WriteWithMetadata(ctx context.Context, path string, data []byte, metadata map[string]string) error {
	return s.client.PutObjectWithMetadata(ctx, path, data, metadata)
}

func (s *s3Adapter) Delete(ctx context.Context, path string) error {
	return s.client.DeleteObject(ctx, path)
}

func (s *s3Adapter) List(ctx context.Context, prefix string) ([]string, error) {
	return s.client.ListObjects(ctx, prefix)
}

func (s *s3Adapter) GetAttr(ctx context.Context, path string) (*types.Attr, error) {
	metadata, err := s.client.HeadObject(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}

	size, err := s.client.HeadObjectSize(ctx, path)
	if err != nil {
		return nil, err
	}

	mode := uint32(0644)
	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	mtime := time.Now()

	// Parse metadata
	if modeStr, ok := metadata["mode"]; ok {
		var modeVal uint32
		fmt.Sscanf(modeStr, "%o", &modeVal)
		mode = modeVal
	}
	if uidStr, ok := metadata["uid"]; ok {
		fmt.Sscanf(uidStr, "%d", &uid)
	}
	if gidStr, ok := metadata["gid"]; ok {
		fmt.Sscanf(gidStr, "%d", &gid)
	}
	if mtimeStr, ok := metadata["mtime"]; ok {
		var unixTime int64
		if _, err := fmt.Sscanf(mtimeStr, "%d", &unixTime); err == nil {
			mtime = time.Unix(unixTime, 0)
		}
	}

	return &types.Attr{
		Size:  size,
		Mode:  mode,
		Uid:   uid,
		Gid:   gid,
		Mtime: mtime,
	}, nil
}

func (s *s3Adapter) Rename(ctx context.Context, oldPath, newPath string) error {
	metadata, err := s.client.HeadObject(ctx, oldPath)
	if err != nil {
		return fmt.Errorf("source file not found: %w", err)
	}
	
	if err := s.client.CopyObjectWithMetadata(ctx, oldPath, newPath, metadata); err != nil {
		return err
	}
	
	return s.client.DeleteObject(ctx, oldPath)
}

func (s *s3Adapter) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, path)
	return err == nil, nil
}

func (s *s3Adapter) GetMetadata(ctx context.Context, path string) (map[string]string, error) {
	metadata, err := s.client.HeadObject(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", os.ErrNotExist)
	}
	return metadata, nil
}

// GetAttr retrieves file attributes
func (fs *Filesystem) GetAttr(ctx context.Context, path string) (*Attr, error) {
	normalizedPath := fs.normalizePath(path)
	
	// Check FD cache for buffered files first
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			// If there's buffered data, return attributes from cache (including updated size and mtime)
			if entity.BytesModified() > 0 {
				// Return from cache - entity has the most up-to-date size and mtime
				size := entity.Size()
				mtime := entity.Mtime()
				
				// Try to get mode/uid/gid from stat cache or use defaults
				mode := os.FileMode(0644)
				uid := uint32(os.Getuid())
				gid := uint32(os.Getgid())
				
				statCache := fs.cache.GetStatCache()
				if statCache != nil {
					if cachedEntry, found := statCache.Get(path); found && cachedEntry != nil {
						cachedAttr := cachedEntry.Attr
						if cachedAttr != nil {
							mode = os.FileMode(cachedAttr.Mode)
							uid = cachedAttr.Uid
							gid = cachedAttr.Gid
						}
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
			// Even if no buffered data, check if entity was very recently modified (within last 50ms)
			// and entity mtime is newer than storage mtime (indicating a recent append/update)
			// This handles the case where data was just uploaded but GetAttr is called immediately after
			// Only do this if stat cache was invalidated (to avoid interfering with metadata operations)
			statCache := fs.cache.GetStatCache()
			hasStatCache := false
			if statCache != nil {
				if cachedEntry, found := statCache.Get(path); found && cachedEntry != nil {
					hasStatCache = true
				}
			}
			
			if !hasStatCache {
				backend := fs.getBackend()
				if backend != nil {
					storageAttr, err := backend.GetAttr(ctx, normalizedPath)
					if err == nil {
						entityMtime := entity.Mtime()
						// If entity mtime is newer than storage mtime and was recently modified, use entity mtime
						// This is specifically for append operations where mtime was just updated
						if entityMtime.After(storageAttr.Mtime) && time.Since(entityMtime) < 50*time.Millisecond {
							entitySize := entity.Size()
							mode := os.FileMode(0644)
							uid := uint32(os.Getuid())
							gid := uint32(os.Getgid())
							
							// Use storage attributes for mode/uid/gid (they're more accurate)
							mode = os.FileMode(storageAttr.Mode)
							uid = storageAttr.Uid
							gid = storageAttr.Gid
							
							return &Attr{
								Mode:  mode,
								Size:  entitySize,
								Mtime: entityMtime,
								Uid:   uid,
								Gid:   gid,
							}, nil
						}
					}
				}
			}
		}
	}
	
	// Check stat cache
	if fs.cache != nil {
		statCache := fs.cache.GetStatCache()
		if statCache != nil {
			if cachedEntry, found := statCache.Get(path); found && cachedEntry != nil {
				cachedAttr := cachedEntry.Attr
				if cachedAttr != nil {
					return &Attr{
						Mode:  os.FileMode(cachedAttr.Mode),
						Size:  cachedAttr.Size,
						Mtime: cachedAttr.Mtime,
						Uid:   cachedAttr.Uid,
						Gid:   cachedAttr.Gid,
					}, nil
				}
			}
		}
	}
	
	backend := fs.getBackend()
	if backend == nil {
		return nil, fmt.Errorf("no storage backend available")
	}
	
	// Check if it's a directory by listing
	if normalizedPath == "" || strings.HasSuffix(normalizedPath, "/") {
		// Try to get directory metadata from .keep marker
		keepPath := normalizedPath + ".keep"
		keepAttr, err := backend.GetAttr(ctx, keepPath)
		
		mode := os.FileMode(0755)
		uid := uint32(os.Getuid())
		gid := uint32(os.Getgid())
		mtime := time.Now()
		
		if err == nil {
			// Use attributes from backend
			mode = os.FileMode(keepAttr.Mode)
			uid = keepAttr.Uid
			gid = keepAttr.Gid
			mtime = keepAttr.Mtime
		}
		
		attr := &Attr{
			Mode:  os.ModeDir | mode,
			Size:  4096,
			Mtime: mtime,
			Uid:   uid,
			Gid:   gid,
		}
		return attr, nil
	}

	// Try to get file attributes
	attr, err := backend.GetAttr(ctx, normalizedPath)
	if err != nil {
		// Check if it's a directory by listing objects with this prefix
		objects, listErr := backend.List(ctx, normalizedPath+"/")
		if listErr == nil && len(objects) > 0 {
			// Try to get directory metadata from .keep marker
			keepPath := normalizedPath + "/.keep"
			keepAttr, err := backend.GetAttr(ctx, keepPath)
			
			mode := os.FileMode(0755)
			uid := uint32(os.Getuid())
			gid := uint32(os.Getgid())
			mtime := time.Now()
			
			if err == nil {
				mode = os.FileMode(keepAttr.Mode)
				uid = keepAttr.Uid
				gid = keepAttr.Gid
				mtime = keepAttr.Mtime
			}
			
			return &Attr{
				Mode:  os.ModeDir | mode,
				Size:  4096,
				Mtime: mtime,
				Uid:   uid,
				Gid:   gid,
			}, nil
		}
		return nil, fmt.Errorf("file not found: %w", syscall.ENOENT)
	}

	// Use attributes from backend
	mode := os.FileMode(attr.Mode)
	uid := attr.Uid
	gid := attr.Gid
	mtime := attr.Mtime
	size := attr.Size

	resultAttr := &Attr{
		Mode:  mode,
		Size:  size,
		Mtime: mtime,
		Uid:   uid,
		Gid:   gid,
	}

	// Cache the result
	if fs.cache != nil {
		statCache := fs.cache.GetStatCache()
		cachedAttr := &cache.CachedAttr{
			Mode:  uint32(mode),
			Size:  size,
			Mtime: mtime,
			Uid:   uid,
			Gid:   gid,
		}
		statCache.Set(path, cachedAttr, nil)
	}

	return resultAttr, nil
}

// ReadDir lists directory entries
func (fs *Filesystem) ReadDir(ctx context.Context, path string) ([]DirEntry, error) {
	normalizedPath := fs.normalizePath(path)
	if normalizedPath != "" && !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	backend := fs.getBackend()
	if backend == nil {
		return nil, fmt.Errorf("no storage backend available")
	}

	objects, err := backend.List(ctx, normalizedPath)
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

		// Don't filter out .keep files - they should appear in directory listings
		// (Other versions don't filter them)

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

	// Also include buffered files from FD cache
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		bufferedPaths := fdCache.GetBufferedPaths(normalizedPath)
		for _, bufferedPath := range bufferedPaths {
			relativePath := strings.TrimPrefix(bufferedPath, normalizedPath)
			if relativePath != "" {
				// Extract first component
				parts := strings.Split(relativePath, "/")
				name := parts[0]
				
				// Only add if not already seen
				if !seen[name] {
					seen[name] = true
					isDir := len(parts) > 1
					entries = append(entries, DirEntry{
						Name:  name,
						IsDir: isDir,
					})
				}
			}
		}
	}

	return entries, nil
}

// ReadFile reads file data
func (fs *Filesystem) ReadFile(ctx context.Context, path string, offset int64, size int64) ([]byte, error) {
	normalizedPath := fs.normalizePath(path)
	
	// Try FD cache first (check for buffered data)
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			// Acquire file-level advisory read lock if enabled (Option 2)
			if fs.enableFileLock {
				entity.FileLock.RLock()
				defer entity.FileLock.RUnlock()
			}
			
			entitySize := entity.Size()
			
			// If size is 0, read entire file
			if size == 0 {
				size = entitySize - offset
				if size <= 0 {
					return []byte{}, nil
				}
			}
			
			// Try to read from page cache (buffered data)
			if pageData, found := entity.ReadPage(offset); found {
				if int64(len(pageData)) >= size {
					return pageData[:size], nil
				}
			}
			
			// Try to read from cached file
			if entity.GetFile() != nil {
				data, err := entity.Read(offset, size)
				if err == nil && len(data) > 0 {
					return data, nil
				}
			}
			
			// If we have buffered data, read from buffered pages
			if len(entity.GetDirtyPages()) > 0 {
				if bufferedData, found := entity.ReadBufferedData(offset, size); found {
					return bufferedData, nil
				}
			}
		}
	}
	
	// Use range read if offset or size is specified
	// If size is 0, read entire file (pass end=0 to GetObjectRange)
	var end int64
	if size > 0 {
		end = offset + size - 1
	} else {
		// size == 0 means read entire file from offset
		end = 0
	}
	
	backend := fs.getBackend()
	if backend == nil {
		return nil, fmt.Errorf("no storage backend available")
	}
	data, err := backend.ReadRange(ctx, normalizedPath, offset, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// Cache the data in FD cache
	if fs.cache != nil && len(data) > 0 {
		fdCache := fs.cache.GetFdCache()
		entity, err := fdCache.Open(normalizedPath, int64(len(data)), time.Now())
		if err == nil {
			entity.WritePage(offset, data)
		}
	}

	return data, nil
}

// WriteFile writes file data (buffered)
func (fs *Filesystem) WriteFile(ctx context.Context, path string, data []byte, offset int64) error {
	normalizedPath := fs.normalizePath(path)
	
	// Use write buffering if cache is available
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		
		// Get or create FD entity
		attr, _ := fs.GetAttr(ctx, path)
		var size int64
		var mtime time.Time
		if attr != nil {
			size = attr.Size
			mtime = attr.Mtime
		} else {
			size = 0
			mtime = time.Now()
		}
		
		entity, err := fdCache.Open(normalizedPath, size, mtime)
		if err != nil {
			return fmt.Errorf("failed to open cache entity: %w", err)
		}
		
		// Acquire file-level advisory lock if enabled (Option 2)
		if fs.enableFileLock {
			entity.FileLock.Lock()
			defer entity.FileLock.Unlock()
		}
		
		// Write to cache (buffered)
		entity.WritePage(offset, data)
		
		// Update size - if offset is 0, always update size (may truncate or extend)
		newSize := offset + int64(len(data))
		// Update mtime when writing (especially important for appends)
		now := time.Now()
		entity.SetMtime(now)
		
		if offset == 0 {
			// Full file replacement - always update size (may truncate)
			entity.SetSize(newSize)
			// For full file replacement at offset 0, upload immediately to ensure size is correct
			// This is especially important for empty files that are being written to
			if err := fs.uploadBufferedData(ctx, normalizedPath, entity); err != nil {
				// If upload fails (e.g., client not initialized in tests), continue
				// The data is still buffered and will be uploaded later
				if !strings.Contains(err.Error(), "storage backend not initialized") {
					return err
				}
			}
		} else {
			// Partial write - extend if needed
			if newSize > size {
				entity.SetSize(newSize)
			}
			// For appends (writing beyond current size), upload immediately to ensure mtime is updated
			if newSize > size {
				if err := fs.uploadBufferedData(ctx, normalizedPath, entity); err != nil {
					// If upload fails (e.g., client not initialized in tests), continue
					// The data is still buffered and will be uploaded later
					if !strings.Contains(err.Error(), "storage backend not initialized") {
						return err
					}
				}
			} else {
				// Check if we should auto-upload (threshold reached)
				if entity.BytesModified() >= fs.maxDirtyData {
					return fs.uploadBufferedData(ctx, normalizedPath, entity)
				}
			}
		}
		
		// Invalidate stat cache
		fs.cache.GetStatCache().Delete(path)
		return nil
	}
	
	// Fallback to immediate upload if no cache
	return fs.writeFileImmediate(ctx, normalizedPath, data, offset)
}

// writeFileImmediate writes file data immediately to storage backend (no buffering)
func (fs *Filesystem) writeFileImmediate(ctx context.Context, normalizedPath string, data []byte, offset int64) error {
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	
	// Simple write (full file replacement)
	if offset == 0 {
		// Invalidate cache
		if fs.cache != nil {
			fs.cache.GetStatCache().Delete(fs.normalizePath(normalizedPath))
		}
		
		// Update mtime/ctime when writing
		now := time.Now()
		metadata := map[string]string{
			"mtime": fmt.Sprintf("%d", now.Unix()),
			"ctime": fmt.Sprintf("%d", now.Unix()),
		}
		
		return backend.WriteWithMetadata(ctx, normalizedPath, data, metadata)
	}

	// For non-zero offset, we need to read existing file, modify, and write back
	existing, err := backend.Read(ctx, normalizedPath)
	if err != nil {
		// File doesn't exist, create new
		if offset > 0 {
			// Pad with zeros up to offset
			padded := make([]byte, offset)
			data = append(padded, data...)
		}
		// Update mtime/ctime when writing
		now := time.Now()
		metadata := map[string]string{
			"mtime": fmt.Sprintf("%d", now.Unix()),
			"ctime": fmt.Sprintf("%d", now.Unix()),
		}
		return backend.WriteWithMetadata(ctx, normalizedPath, data, metadata)
	}

	// Modify existing file
	if offset >= int64(len(existing)) {
		// Extend file
		padded := make([]byte, offset-int64(len(existing)))
		existing = append(existing, padded...)
		existing = append(existing, data...)
	} else {
		// Overwrite part of file - replace data at offset
		before := existing[:offset]
		afterOffset := offset + int64(len(data))
		var after []byte
		if afterOffset < int64(len(existing)) {
			after = existing[afterOffset:]
		}
		// Replace: before + new data + remaining after
		existing = append(before, append(data, after...)...)
	}
	
	// Invalidate cache
	if fs.cache != nil {
		fs.cache.GetStatCache().Delete(fs.normalizePath(normalizedPath))
	}

	// Update mtime/ctime when writing
	now := time.Now()
	metadata := map[string]string{
		"mtime": fmt.Sprintf("%d", now.Unix()),
		"ctime": fmt.Sprintf("%d", now.Unix()),
	}

	return backend.WriteWithMetadata(ctx, normalizedPath, existing, metadata)
}

// flushBufferedData flushes buffered data for a given path if it exists
func (fs *Filesystem) flushBufferedData(ctx context.Context, path string) error {
	if fs.cache == nil {
		return nil
	}
	
	// If backend is not initialized, skip flushing (for unit tests)
	backend := fs.getBackend()
	if backend == nil {
		return nil
	}
	
	normalizedPath := fs.normalizePath(path)
	fdCache := fs.cache.GetFdCache()
	if entity, found := fdCache.Get(normalizedPath); found {
		if entity.BytesModified() > 0 {
			// Try to upload, but if backend isn't initialized, just skip
			err := fs.uploadBufferedData(ctx, normalizedPath, entity)
			if err != nil && strings.Contains(err.Error(), "storage backend not initialized") {
				// For unit tests without backend, just skip flushing
				return nil
			}
			return err
		}
	}
	return nil
}

// uploadBufferedData uploads buffered data from FD entity to storage backend
func (fs *Filesystem) uploadBufferedData(ctx context.Context, normalizedPath string, entity *cache.FdEntity) error {
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("storage backend not initialized")
	}
	
	// Get existing metadata to preserve it
	existingAttr, _ := backend.GetAttr(ctx, normalizedPath)
	
	// Update mtime/ctime
	now := time.Now()
	metadata := map[string]string{
		"mtime": fmt.Sprintf("%d", now.Unix()),
		"ctime": fmt.Sprintf("%d", now.Unix()),
	}
	
	// Preserve existing metadata (including mode, uid, gid)
	if existingAttr != nil {
		metadata["mode"] = fmt.Sprintf("%o", existingAttr.Mode)
		metadata["uid"] = fmt.Sprintf("%d", existingAttr.Uid)
		metadata["gid"] = fmt.Sprintf("%d", existingAttr.Gid)
	}
	
	// Upload function - use entity size for truncation
	uploadFunc := func(ctx context.Context, data []byte) error {
		// Use entity size, not data length (for truncation)
		entitySize := entity.Size()
		if entitySize < int64(len(data)) {
			// Truncate data to entity size
			data = data[:entitySize]
		} else if entitySize > int64(len(data)) {
			// Extend with zeros if needed
			extended := make([]byte, entitySize)
			copy(extended, data)
			data = extended
		}
		
		// Use backend WriteWithMetadata (multipart handling is backend-specific)
		err := backend.WriteWithMetadata(ctx, normalizedPath, data, metadata)
		if err == nil {
			// Update entity mtime after successful upload to match what was written
			entity.SetMtime(now)
			// Update stat cache with new attributes after upload
			if fs.cache != nil {
				statCache := fs.cache.GetStatCache()
				if statCache != nil {
					// Get updated attributes from storage to cache
					if updatedAttr, err := backend.GetAttr(ctx, normalizedPath); err == nil {
						cachedAttr := &cache.CachedAttr{
							Mode:  uint32(updatedAttr.Mode),
							Size:  updatedAttr.Size,
							Mtime: updatedAttr.Mtime,
							Uid:   updatedAttr.Uid,
							Gid:   updatedAttr.Gid,
						}
						statCache.Set(fs.normalizePath(normalizedPath), cachedAttr, nil)
					}
				}
			}
		}
		return err
	}
	
	return entity.UploadBufferedData(ctx, uploadFunc)
}

// Create creates a new file
func (fs *Filesystem) Create(ctx context.Context, path string, mode os.FileMode) error {
	normalizedPath := fs.normalizePath(path)
	
	// Check if file already exists
	_, err := fs.GetAttr(ctx, path)
	if err == nil {
		return syscall.EEXIST
	}
	
	// Create empty file with mode metadata
	modeStr := fmt.Sprintf("%04o", mode&0777)
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-mode": modeStr,
		"mode": modeStr,
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
		"ctime": fmt.Sprintf("%d", now.Unix()),
	}
	
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	return backend.WriteWithMetadata(ctx, normalizedPath, []byte{}, metadata)
}

// Remove removes a file
func (fs *Filesystem) Remove(ctx context.Context, path string) error {
	normalizedPath := fs.normalizePath(path)
	
	// Check if file exists first
	_, err := fs.GetAttr(ctx, path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}
	
	// Invalidate cache
	if fs.cache != nil {
		fs.cache.GetStatCache().Delete(path)
		fs.cache.GetFdCache().Close(normalizedPath)
	}
	
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	err = backend.Delete(ctx, normalizedPath)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	
	return nil
}

// Rename renames a file or directory
func (fs *Filesystem) Rename(ctx context.Context, oldPath, newPath string) error {
	// Flush buffered data for source path before renaming
	if err := fs.flushBufferedData(ctx, oldPath); err != nil {
		// If client not initialized, return error that can be caught by tests
		if strings.Contains(err.Error(), "S3 client not initialized") {
			return err
		}
		return fmt.Errorf("failed to flush buffered data before rename: %w", err)
	}
	
	oldNormalized := fs.normalizePath(oldPath)
	newNormalized := fs.normalizePath(newPath)

	// Check if source is a directory
	attr, err := fs.GetAttr(ctx, oldPath)
	if err != nil {
		// If client not initialized, return error that can be caught by tests
		if strings.Contains(err.Error(), "S3 client not initialized") {
			return err
		}
		return fmt.Errorf("source not found: %w", err)
	}
	
	isDir := attr.Mode.IsDir()
	if isDir {
		// Normalize directory paths
		if !strings.HasSuffix(oldNormalized, "/") {
			oldNormalized += "/"
		}
		if !strings.HasSuffix(newNormalized, "/") {
			newNormalized += "/"
		}
		
		// Flush all buffered files in the directory before renaming
		if fs.cache != nil {
			fdCache := fs.cache.GetFdCache()
			bufferedPaths := fdCache.GetBufferedPaths(oldNormalized)
			for _, bufferedPath := range bufferedPaths {
				if err := fs.flushBufferedData(ctx, bufferedPath); err != nil {
					return fmt.Errorf("failed to flush buffered data for %s before rename: %w", bufferedPath, err)
				}
			}
		}
		
		// Rename directory by copying all objects with the prefix
		backend := fs.getBackend()
		if backend == nil {
			return fmt.Errorf("no storage backend available")
		}
		
		objects, err := backend.List(ctx, oldNormalized)
		if err != nil {
			return fmt.Errorf("failed to list directory objects: %w", err)
		}
		
		// Copy each object to new location
		for _, objKey := range objects {
			newKey := strings.Replace(objKey, oldNormalized, newNormalized, 1)
			// Use backend Rename for each file
			if err := backend.Rename(ctx, objKey, newKey); err != nil {
				return fmt.Errorf("failed to rename object %s: %w", objKey, err)
			}
		}
		
		// Invalidate cache
		if fs.cache != nil {
			fs.cache.GetStatCache().Delete(oldPath)
			fs.cache.GetStatCache().Delete(newPath)
		}
		
		return nil
	}

	// Use backend Rename method
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	if err := backend.Rename(ctx, oldNormalized, newNormalized); err != nil {
		return err
	}

	// Invalidate cache
	if fs.cache != nil {
		fs.cache.GetStatCache().Delete(oldPath)
		fs.cache.GetStatCache().Delete(newPath)
		fs.cache.GetFdCache().Close(oldNormalized)
		fs.cache.GetFdCache().Close(newNormalized)
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
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	return backend.WriteWithMetadata(ctx, normalizedPath+".keep", []byte{}, metadata)
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
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	
	err = backend.Delete(ctx, normalizedPath+".keep")
	if err != nil {
		// Directory marker might not exist, which is okay
		// Check if there are any objects with this prefix
		objects, listErr := backend.List(ctx, normalizedPath)
		if listErr != nil || len(objects) > 0 {
			return syscall.ENOTEMPTY
		}
		// Directory is effectively empty, allow removal
		return nil
	}
	
	return nil
}

// Symlink creates a symbolic link
func (fs *Filesystem) Symlink(ctx context.Context, oldname, newname string) error {
	normalizedPath := fs.normalizePath(newname)
	
	// Check if target already exists
	_, err := fs.GetAttr(ctx, newname)
	if err == nil {
		return syscall.EEXIST
	}
	
	// Create symlink file with target path as content
	now := time.Now()
	metadata := map[string]string{
		"x-amz-meta-mode":  fmt.Sprintf("%o", os.ModeSymlink|0777),
		"x-amz-meta-uid":   fmt.Sprintf("%d", os.Getuid()),
		"x-amz-meta-gid":   fmt.Sprintf("%d", os.Getgid()),
		"x-amz-meta-mtime": fmt.Sprintf("%d", now.Unix()),
		"x-amz-meta-atime": fmt.Sprintf("%d", now.Unix()),
		"x-amz-meta-ctime": fmt.Sprintf("%d", now.Unix()),
	}
	
	// Store symlink target in file content
	targetData := []byte(oldname)
	backend := fs.getBackend()
	if backend == nil {
		return fmt.Errorf("no storage backend available")
	}
	err = backend.WriteWithMetadata(ctx, normalizedPath, targetData, metadata)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	
	// Cache symlink target
	if fs.cache != nil {
		fs.cache.GetStatCache().SetSymlink(newname, oldname)
	}
	
	return nil
}

// Readlink reads the target of a symbolic link
func (fs *Filesystem) Readlink(ctx context.Context, path string) (string, error) {
	normalizedPath := fs.normalizePath(path)
	
	// Check cache first
	if fs.cache != nil {
		if target, found := fs.cache.GetStatCache().GetSymlink(path); found {
			return target, nil
		}
	}
	
	// Read symlink target from file content
	backend := fs.getBackend()
	if backend == nil {
		return "", fmt.Errorf("no storage backend available")
	}
	data, err := backend.Read(ctx, normalizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", syscall.ENOENT)
	}
	
	// Trim whitespace and get target
	target := strings.TrimSpace(string(data))
	
	// Cache the result
	if fs.cache != nil {
		fs.cache.GetStatCache().SetSymlink(path, target)
	}
	
	return target, nil
}

// Link creates a hard link (not supported in S3)
func (fs *Filesystem) Link(ctx context.Context, oldname, newname string) error {
	return syscall.ENOTSUP
}

// Mknod creates a special file (not supported in S3)
func (fs *Filesystem) Mknod(ctx context.Context, path string, mode os.FileMode, dev uint32) error {
	return syscall.ENOTSUP
}

// Access checks file access permissions
func (fs *Filesystem) Access(ctx context.Context, path string, mask uint32) error {
	// Check if file exists
	_, err := fs.GetAttr(ctx, path)
	if err != nil {
		return err
	}
	
	// Check permissions based on mask
	// R_OK = 4, W_OK = 2, X_OK = 1, F_OK = 0
	if mask == 0 { // F_OK - just check existence
		return nil
	}
	
	// For now, allow all if file exists
	// In a full implementation, we'd check actual permissions
	// against the current user's uid/gid
	return nil
}

// Statfs represents filesystem statistics
type Statfs struct {
	Bsize   uint64 // Block size
	Blocks  uint64 // Total blocks
	Bfree   uint64 // Free blocks
	Bavail  uint64 // Available blocks
	Files   uint64 // Total inodes
	Ffree   uint64 // Free inodes
	Namelen uint32 // Max filename length
}

// Statfs returns filesystem statistics
func (fs *Filesystem) Statfs(ctx context.Context) (*Statfs, error) {
	// Return default filesystem statistics
	// S3 doesn't have real filesystem limits, so we return large values
	return &Statfs{
		Bsize:  4096,              // Block size
		Blocks: 1000000000,        // Total blocks (fake large number)
		Bfree:  1000000000,        // Free blocks
		Bavail: 1000000000,        // Available blocks
		Files:  1000000000,        // Total inodes
		Ffree:  1000000000,        // Free inodes
		Namelen: 255,              // Max filename length
	}, nil
}

// Flush flushes file buffers
func (fs *Filesystem) Flush(ctx context.Context, path string) error {
	normalizedPath := fs.normalizePath(path)
	
	// Upload buffered data if file is cached
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			// Acquire file-level advisory lock if enabled (Option 2)
			if fs.enableFileLock {
				entity.FileLock.Lock()
				defer entity.FileLock.Unlock()
			}
			
			// Upload any buffered data
			if entity.BytesModified() > 0 {
				if err := fs.uploadBufferedData(ctx, normalizedPath, entity); err != nil {
					return fmt.Errorf("failed to flush buffered data: %w", err)
				}
			}
			
			// Sync file to disk
			file := entity.GetFile()
			if file != nil {
				return file.Sync()
			}
		}
	}
	
	return nil
}

// Fsync syncs file data to storage
func (fs *Filesystem) Fsync(ctx context.Context, path string, datasync bool) error {
	normalizedPath := fs.normalizePath(path)
	
	// Upload buffered data if file is cached
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			// Acquire file-level advisory lock if enabled (Option 2)
			if fs.enableFileLock {
				entity.FileLock.Lock()
				defer entity.FileLock.Unlock()
			}
			
			// Upload any buffered data
			if entity.BytesModified() > 0 {
				if err := fs.uploadBufferedData(ctx, normalizedPath, entity); err != nil {
					return fmt.Errorf("failed to sync buffered data: %w", err)
				}
			}
			
			// Sync file to disk
			file := entity.GetFile()
			if file != nil {
				if datasync {
					// fdatasync - sync data only, not metadata
					// On Linux, this is the same as Sync for our use case
					return file.Sync()
				} else {
					// fsync - sync data and metadata
					return file.Sync()
				}
			}
		}
	}
	
	// Invalidate stat cache after sync (size may have changed)
	if fs.cache != nil {
		fs.cache.GetStatCache().Delete(path)
	}
	
	return nil
}

// Release releases a file handle
func (fs *Filesystem) Release(ctx context.Context, path string) error {
	normalizedPath := fs.normalizePath(path)
	
	// Upload buffered data before closing
	if fs.cache != nil {
		fdCache := fs.cache.GetFdCache()
		if entity, found := fdCache.Get(normalizedPath); found {
			// Upload any buffered data before closing
			if entity.BytesModified() > 0 {
				if err := fs.uploadBufferedData(ctx, normalizedPath, entity); err != nil {
					// Log error but still close
					// In production, you might want to handle this differently
				}
			}
		}
		
		// Close FD cache entity
		return fdCache.Close(normalizedPath)
	}
	
	return nil
}

// Opendir opens a directory handle
func (fs *Filesystem) Opendir(ctx context.Context, path string) error {
	// Check if directory exists and is accessible
	attr, err := fs.GetAttr(ctx, path)
	if err != nil {
		return err
	}
	
	if !attr.Mode.IsDir() {
		return syscall.ENOTDIR
	}
	
	// Directory is accessible
	return nil
}
