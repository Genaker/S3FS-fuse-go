package cache

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// FdEntity represents a cached file descriptor entity
type FdEntity struct {
	mu            sync.RWMutex // Entity-level mutex (always used)
	FileLock      sync.RWMutex // File-level advisory lock (optional, for stricter coordination)
	path          string
	file          *os.File
	size          int64
	mtime         time.Time
	refCount      int
	lastAccess    time.Time
	pages         map[int64]*Page // Page cache: offset -> page data
	pageSize      int64
	bytesModified int64          // Total bytes modified but not yet uploaded
	dirtyPages    map[int64]bool // Track which pages are dirty (not uploaded)
}

// Page represents a cached page of file data
type Page struct {
	Offset     int64
	Data       []byte
	Size       int64
	Dirty      bool
	LastAccess time.Time
}

// FdInfo contains metadata about a file descriptor
type FdInfo struct {
	Path       string
	Size       int64
	Mtime      time.Time
	RefCount   int
	LastAccess time.Time
}

// FdCacheManager manages file descriptor cache
type FdCacheManager struct {
	mu            sync.RWMutex
	entities      map[string]*FdEntity
	maxSize       int
	maxOpenFiles  int
	pageSize      int64
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewFdCacheManager creates a new FD cache manager
func NewFdCacheManager(maxSize int, maxOpenFiles int, pageSize int64) *FdCacheManager {
	fcm := &FdCacheManager{
		entities:     make(map[string]*FdEntity),
		maxSize:      maxSize,
		maxOpenFiles: maxOpenFiles,
		pageSize:     pageSize,
		stopCleanup:  make(chan struct{}),
	}

	// Start cleanup goroutine
	fcm.cleanupTicker = time.NewTicker(30 * time.Second)
	go fcm.cleanupUnused()

	return fcm
}

// Open opens or retrieves a cached file entity
func (fcm *FdCacheManager) Open(path string, size int64, mtime time.Time) (*FdEntity, error) {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	entity, exists := fcm.entities[path]
	if exists {
		entity.mu.Lock()
		entity.refCount++
		entity.lastAccess = time.Now()
		entity.mu.Unlock()
		return entity, nil
	}

	// Check if we've reached max open files
	if len(fcm.entities) >= fcm.maxOpenFiles {
		fcm.closeOldest()
	}

	// Create new entity
	entity = &FdEntity{
		path:          path,
		size:          size,
		mtime:         mtime,
		refCount:      1,
		lastAccess:    time.Now(),
		pages:         make(map[int64]*Page),
		pageSize:      fcm.pageSize,
		bytesModified: 0,
		dirtyPages:    make(map[int64]bool),
	}

	fcm.entities[path] = entity
	return entity, nil
}

// Get retrieves a cached entity without incrementing ref count
func (fcm *FdCacheManager) Get(path string) (*FdEntity, bool) {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	entity, exists := fcm.entities[path]
	if !exists {
		return nil, false
	}

	entity.mu.RLock()
	defer entity.mu.RUnlock()
	entity.lastAccess = time.Now()
	return entity, true
}

// Close closes a file entity and decrements ref count
func (fcm *FdCacheManager) Close(path string) error {
	fcm.mu.Lock()
	defer fcm.mu.Unlock()

	entity, exists := fcm.entities[path]
	if !exists {
		return nil
	}

	entity.mu.Lock()
	entity.refCount--
	if entity.refCount <= 0 {
		if entity.file != nil {
			entity.file.Close()
			entity.file = nil
		}
		delete(fcm.entities, path)
	}
	entity.mu.Unlock()

	return nil
}

// GetInfo returns information about a cached entity
func (fcm *FdCacheManager) GetInfo(path string) (*FdInfo, bool) {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	entity, exists := fcm.entities[path]
	if !exists {
		return nil, false
	}

	entity.mu.RLock()
	defer entity.mu.RUnlock()

	return &FdInfo{
		Path:       entity.path,
		Size:       entity.size,
		Mtime:      entity.mtime,
		RefCount:   entity.refCount,
		LastAccess: entity.lastAccess,
	}, true
}

// HasOpenEntity checks if an entity is open
func (fcm *FdCacheManager) HasOpenEntity(path string) bool {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	entity, exists := fcm.entities[path]
	if !exists {
		return false
	}

	entity.mu.RLock()
	defer entity.mu.RUnlock()
	return entity.refCount > 0
}

// GetOpenFdCount returns the number of open file descriptors
func (fcm *FdCacheManager) GetOpenFdCount(path string) int {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	entity, exists := fcm.entities[path]
	if !exists {
		return 0
	}

	entity.mu.RLock()
	defer entity.mu.RUnlock()
	return entity.refCount
}

// closeOldest closes the oldest unused entity
func (fcm *FdCacheManager) closeOldest() {
	var oldestPath string
	var oldestTime time.Time
	var oldestEntity *FdEntity

	for path, entity := range fcm.entities {
		entity.mu.RLock()
		if entity.refCount == 0 {
			if oldestPath == "" || entity.lastAccess.Before(oldestTime) {
				oldestPath = path
				oldestTime = entity.lastAccess
				oldestEntity = entity
			}
		}
		entity.mu.RUnlock()
	}

	if oldestEntity != nil {
		oldestEntity.mu.Lock()
		if oldestEntity.file != nil {
			oldestEntity.file.Close()
			oldestEntity.file = nil
		}
		oldestEntity.mu.Unlock()
		delete(fcm.entities, oldestPath)
	}
}

// cleanupUnused periodically cleans up unused entities
func (fcm *FdCacheManager) cleanupUnused() {
	for {
		select {
		case <-fcm.cleanupTicker.C:
			fcm.mu.Lock()
			now := time.Now()
			expired := time.Hour // Entities unused for 1 hour are expired

			for path, entity := range fcm.entities {
				entity.mu.RLock()
				if entity.refCount == 0 && now.Sub(entity.lastAccess) > expired {
					entity.mu.RUnlock()
					entity.mu.Lock()
					if entity.file != nil {
						entity.file.Close()
						entity.file = nil
					}
					entity.mu.Unlock()
					delete(fcm.entities, path)
				} else {
					entity.mu.RUnlock()
				}
			}
			fcm.mu.Unlock()
		case <-fcm.stopCleanup:
			return
		}
	}
}

// CloseAll stops the cleanup goroutine and closes all entities
func (fcm *FdCacheManager) CloseAll() {
	if fcm.cleanupTicker != nil {
		fcm.cleanupTicker.Stop()
	}
	close(fcm.stopCleanup)

	// Close all entities
	fcm.mu.Lock()
	defer fcm.mu.Unlock()
	for _, entity := range fcm.entities {
		entity.mu.Lock()
		if entity.file != nil {
			entity.file.Close()
			entity.file = nil
		}
		entity.mu.Unlock()
	}
	fcm.entities = make(map[string]*FdEntity)
}

// GetBufferedPaths returns all paths that have buffered data
func (fcm *FdCacheManager) GetBufferedPaths(prefix string) []string {
	fcm.mu.RLock()
	defer fcm.mu.RUnlock()

	var paths []string
	for path, entity := range fcm.entities {
		if strings.HasPrefix(path, prefix) && entity.BytesModified() > 0 {
			paths = append(paths, path)
		}
	}
	return paths
}

// ReadPage reads a page from cache or returns nil if not cached
func (fe *FdEntity) ReadPage(offset int64) ([]byte, bool) {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	pageOffset := (offset / fe.pageSize) * fe.pageSize
	page, exists := fe.pages[pageOffset]
	if !exists {
		return nil, false
	}

	// Check if requested offset is within page
	if offset < pageOffset || offset >= pageOffset+int64(len(page.Data)) {
		return nil, false
	}

	// Return data starting from offset
	pageStart := offset - pageOffset
	page.LastAccess = time.Now()
	return page.Data[pageStart:], true
}

// WritePage writes a page to cache
func (fe *FdEntity) WritePage(offset int64, data []byte) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	pageOffset := (offset / fe.pageSize) * fe.pageSize
	offsetInPage := offset - pageOffset
	endOffset := offset + int64(len(data))
	pageEndOffset := pageOffset + fe.pageSize
	if endOffset > pageEndOffset {
		endOffset = pageEndOffset
	}
	pageDataSize := endOffset - pageOffset

	// Truncate cache if needed (before adding new page)
	if len(fe.pages) >= 100 { // Max 100 pages per entity
		fe.evictOldestPage()
	}

	// Check if page already exists
	existingPage, exists := fe.pages[pageOffset]
	var pageData []byte

	if exists {
		// Merge with existing page data
		existingSize := int64(len(existingPage.Data))
		if existingSize < pageDataSize {
			// Extend page data
			extended := make([]byte, pageDataSize)
			copy(extended, existingPage.Data)
			pageData = extended
		} else {
			pageData = make([]byte, existingSize)
			copy(pageData, existingPage.Data)
		}

		// Update bytesModified: subtract old dirty size
		if existingPage.Dirty {
			fe.bytesModified -= existingPage.Size
		}
	} else {
		// Create new page data
		pageData = make([]byte, pageDataSize)
	}

	// Write new data into page at correct offset
	copy(pageData[offsetInPage:], data)

	page := &Page{
		Offset:     pageOffset,
		Data:       pageData,
		Size:       int64(len(pageData)),
		Dirty:      true,
		LastAccess: time.Now(),
	}

	fe.pages[pageOffset] = page
	fe.dirtyPages[pageOffset] = true
	fe.bytesModified += page.Size

	// If we're still over limit after adding, evict more
	for len(fe.pages) > 100 {
		fe.evictOldestPage()
	}
}

// BytesModified returns the number of bytes modified but not uploaded
func (fe *FdEntity) BytesModified() int64 {
	fe.mu.RLock()
	defer fe.mu.RUnlock()
	return fe.bytesModified
}

// MarkPageClean marks a page as clean (uploaded)
func (fe *FdEntity) MarkPageClean(pageOffset int64) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	if page, exists := fe.pages[pageOffset]; exists {
		if page.Dirty {
			page.Dirty = false
			fe.bytesModified -= page.Size
			if fe.bytesModified < 0 {
				fe.bytesModified = 0
			}
		}
	}
	delete(fe.dirtyPages, pageOffset)
}

// GetDirtyPages returns all dirty page offsets
func (fe *FdEntity) GetDirtyPages() []int64 {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	dirty := make([]int64, 0, len(fe.dirtyPages))
	for offset := range fe.dirtyPages {
		dirty = append(dirty, offset)
	}
	return dirty
}

// ReadBufferedData reads data from buffered pages and cached file
func (fe *FdEntity) ReadBufferedData(offset int64, size int64) ([]byte, bool) {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	entitySize := fe.size
	if size == 0 {
		size = entitySize - offset
		if size <= 0 {
			return []byte{}, true
		}
	}

	// Create buffer
	maxOffset := offset + size
	if maxOffset > entitySize {
		maxOffset = entitySize
	}
	if maxOffset <= offset {
		return []byte{}, true
	}

	bufferedData := make([]byte, maxOffset)

	// Read existing file if available
	if fe.file != nil {
		fe.file.Seek(0, 0)
		existingData, _ := io.ReadAll(fe.file)
		if len(existingData) > 0 {
			copy(bufferedData, existingData)
		}
	}

	// Overlay dirty pages
	for pageOffset := range fe.dirtyPages {
		if page, exists := fe.pages[pageOffset]; exists {
			end := pageOffset + page.Size
			if end > int64(len(bufferedData)) {
				end = int64(len(bufferedData))
			}
			if pageOffset < int64(len(bufferedData)) {
				copy(bufferedData[pageOffset:end], page.Data[:end-pageOffset])
			}
		}
	}

	// Return requested range
	if offset < int64(len(bufferedData)) {
		end := offset + size
		if end > int64(len(bufferedData)) {
			end = int64(len(bufferedData))
		}
		if offset < end {
			return bufferedData[offset:end], true
		}
	}

	return nil, false
}

// evictOldestPage removes the oldest page from cache
func (fe *FdEntity) evictOldestPage() {
	var oldestOffset int64
	var oldestTime time.Time

	for offset, page := range fe.pages {
		if oldestOffset == 0 || page.LastAccess.Before(oldestTime) {
			oldestOffset = offset
			oldestTime = page.LastAccess
		}
	}

	if oldestOffset != 0 {
		delete(fe.pages, oldestOffset)
	}
}

// GetFile returns the underlying file handle (returns nil if not set)
func (fe *FdEntity) GetFile() *os.File {
	fe.mu.RLock()
	defer fe.mu.RUnlock()
	return fe.file
}

// SetFileFromTemp creates a temporary file and sets it
func (fe *FdEntity) SetFileFromTemp() (*os.File, error) {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	if fe.file != nil {
		return fe.file, nil
	}

	// Create temporary file for caching
	tmpFile, err := os.CreateTemp("", "s3fs-cache-*")
	if err != nil {
		return nil, err
	}

	fe.file = tmpFile
	return fe.file, nil
}

// SetFile sets the underlying file handle
func (fe *FdEntity) SetFile(file *os.File) {
	fe.mu.Lock()
	defer fe.mu.Unlock()
	fe.file = file
}

// Read reads data from the cached file
func (fe *FdEntity) Read(offset int64, size int64) ([]byte, error) {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	if fe.file == nil {
		return nil, io.EOF
	}

	_, err := fe.file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	data := make([]byte, size)
	n, err := fe.file.Read(data)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return data[:n], nil
}

// Write writes data to the cached file
func (fe *FdEntity) Write(offset int64, data []byte) error {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	if fe.file == nil {
		return io.ErrClosedPipe
	}

	_, err := fe.file.Seek(offset, 0)
	if err != nil {
		return err
	}

	_, err = fe.file.Write(data)
	if err != nil {
		return err
	}

	fe.lastAccess = time.Now()
	return nil
}

// Sync syncs the cached file to disk
func (fe *FdEntity) Sync() error {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	if fe.file == nil {
		return nil
	}

	return fe.file.Sync()
}

// Size returns the file size
func (fe *FdEntity) Size() int64 {
	fe.mu.RLock()
	defer fe.mu.RUnlock()
	return fe.size
}

// SetSize updates the file size
func (fe *FdEntity) SetSize(size int64) {
	fe.mu.Lock()
	defer fe.mu.Unlock()
	fe.size = size
}

func (fe *FdEntity) Mtime() time.Time {
	fe.mu.RLock()
	defer fe.mu.RUnlock()
	return fe.mtime
}

func (fe *FdEntity) SetMtime(mtime time.Time) {
	fe.mu.Lock()
	defer fe.mu.Unlock()
	fe.mtime = mtime
}

// UploadBufferedData uploads all dirty pages to S3 using the provided upload function
func (fe *FdEntity) UploadBufferedData(ctx context.Context, uploadFunc func(ctx context.Context, data []byte) error) error {
	fe.mu.Lock()

	// Get dirty data
	dirtyPages := make([]int64, 0, len(fe.dirtyPages))
	for offset := range fe.dirtyPages {
		dirtyPages = append(dirtyPages, offset)
	}

	if len(dirtyPages) == 0 {
		fe.mu.Unlock()
		return nil
	}

	// Use entity size, not max offset from dirty pages
	entitySize := fe.size

	// Sort offsets
	sort.Slice(dirtyPages, func(i, j int) bool {
		return dirtyPages[i] < dirtyPages[j]
	})

	// Read existing file if it exists, or create new buffer
	var fullData []byte
	if fe.file != nil {
		// Read entire file
		fe.file.Seek(0, 0)
		fileData, err := io.ReadAll(fe.file)
		if err == nil {
			fullData = fileData
		}
	}

	// Ensure buffer matches entity size
	if int64(len(fullData)) < entitySize {
		// Extend with zeros
		extended := make([]byte, entitySize)
		copy(extended, fullData)
		fullData = extended
	} else if len(fullData) == 0 {
		fullData = make([]byte, entitySize)
	} else if int64(len(fullData)) > entitySize {
		// Truncate to entity size
		fullData = fullData[:entitySize]
	}

	// Write dirty pages into buffer
	for _, offset := range dirtyPages {
		if page, exists := fe.pages[offset]; exists {
			// Ensure we don't go out of bounds
			pageEnd := offset + page.Size
			if pageEnd > entitySize {
				pageEnd = entitySize
			}
			if offset < entitySize {
				copy(fullData[offset:pageEnd], page.Data[:pageEnd-offset])
			}
		}
	}

	fe.mu.Unlock()

	// Upload data
	if err := uploadFunc(ctx, fullData); err != nil {
		return err
	}

	// Mark all pages as clean
	fe.mu.Lock()
	defer fe.mu.Unlock()
	for _, offset := range dirtyPages {
		if page, exists := fe.pages[offset]; exists {
			page.Dirty = false
		}
		delete(fe.dirtyPages, offset)
	}
	fe.bytesModified = 0

	return nil
}
