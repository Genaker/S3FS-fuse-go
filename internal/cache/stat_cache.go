package cache

import (
	"sync"
	"time"
)

// StatCacheEntry represents a cached stat entry
type StatCacheEntry struct {
	Path      string
	Attr      *CachedAttr
	Metadata  map[string]string
	Symlink   string // For symlink cache
	ExpiresAt time.Time
	LastAccess time.Time
}

// CachedAttr represents cached file attributes
type CachedAttr struct {
	Mode  uint32
	Size  int64
	Mtime time.Time
	Uid   uint32
	Gid   uint32
}

// StatCache manages cached file attributes
type StatCache struct {
	mu            sync.RWMutex
	entries       map[string]*StatCacheEntry
	maxSize       int
	defaultTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewStatCache creates a new stat cache
func NewStatCache(maxSize int, defaultTTL time.Duration) *StatCache {
	sc := &StatCache{
		entries:    make(map[string]*StatCacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	sc.cleanupTicker = time.NewTicker(defaultTTL / 2)
	go sc.cleanupExpired()

	return sc
}

// Get retrieves a cached stat entry
func (sc *StatCache) Get(path string) (*StatCacheEntry, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, exists := sc.entries[path]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	// Update last access time
	entry.LastAccess = time.Now()
	return entry, true
}

// Set stores a stat entry in cache
func (sc *StatCache) Set(path string, attr *CachedAttr, metadata map[string]string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Truncate cache if needed
	sc.truncateIfNeeded()

	entry := &StatCacheEntry{
		Path:      path,
		Attr:      attr,
		Metadata:  metadata,
		ExpiresAt: time.Now().Add(sc.defaultTTL),
		LastAccess: time.Now(),
	}

	sc.entries[path] = entry
}

// SetSymlink stores a symlink target in cache
func (sc *StatCache) SetSymlink(path string, target string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Truncate cache if needed
	sc.truncateIfNeeded()

	entry := &StatCacheEntry{
		Path:      path,
		Symlink:   target,
		ExpiresAt: time.Now().Add(sc.defaultTTL),
		LastAccess: time.Now(),
	}

	sc.entries[path] = entry
}

// GetSymlink retrieves a cached symlink target
func (sc *StatCache) GetSymlink(path string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, exists := sc.entries[path]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	if entry.Symlink == "" {
		return "", false
	}

	// Update last access time
	entry.LastAccess = time.Now()
	return entry.Symlink, true
}

// Delete removes an entry from cache
func (sc *StatCache) Delete(path string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.entries, path)
}

// Clear removes all entries from cache
func (sc *StatCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries = make(map[string]*StatCacheEntry)
}

// Size returns the current number of cached entries
func (sc *StatCache) Size() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.entries)
}

// SetMaxSize updates the maximum cache size
func (sc *StatCache) SetMaxSize(maxSize int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.maxSize = maxSize
	sc.truncateIfNeeded()
}

// SetTTL updates the default TTL
func (sc *StatCache) SetTTL(ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.defaultTTL = ttl
}

// truncateIfNeeded removes oldest entries if cache exceeds max size
func (sc *StatCache) truncateIfNeeded() {
	if len(sc.entries) < sc.maxSize {
		return
	}

	// Find entries to remove (oldest last access time)
	type entryWithTime struct {
		path       string
		lastAccess time.Time
	}

	entries := make([]entryWithTime, 0, len(sc.entries))
	for path, entry := range sc.entries {
		entries = append(entries, entryWithTime{
			path:       path,
			lastAccess: entry.LastAccess,
		})
	}

	// Sort by last access time (oldest first)
	// Simple selection sort for small caches
	for i := 0; i < len(entries)-1; i++ {
		minIdx := i
		for j := i + 1; j < len(entries); j++ {
			if entries[j].lastAccess.Before(entries[minIdx].lastAccess) {
				minIdx = j
			}
		}
		entries[i], entries[minIdx] = entries[minIdx], entries[i]
	}

	// Remove oldest entries
	toRemove := len(sc.entries) - sc.maxSize + 1
	for i := 0; i < toRemove && i < len(entries); i++ {
		delete(sc.entries, entries[i].path)
	}
}

// cleanupExpired periodically removes expired entries
func (sc *StatCache) cleanupExpired() {
	for {
		select {
		case <-sc.cleanupTicker.C:
			sc.mu.Lock()
			now := time.Now()
			for path, entry := range sc.entries {
				if now.After(entry.ExpiresAt) {
					delete(sc.entries, path)
				}
			}
			sc.mu.Unlock()
		case <-sc.stopCleanup:
			return
		}
	}
}

// Close stops the cleanup goroutine
func (sc *StatCache) Close() {
	if sc.cleanupTicker != nil {
		sc.cleanupTicker.Stop()
	}
	close(sc.stopCleanup)
}
