package cache

import (
	"time"
)

// Manager combines stat cache and FD cache
type Manager struct {
	statCache *StatCache
	fdCache   *FdCacheManager
	tree      *CacheTree
}

// NewManager creates a new cache manager
func NewManager(statMaxSize int, statTTL time.Duration, fdMaxSize int, fdMaxOpenFiles int, pageSize int64) *Manager {
	return &Manager{
		statCache: NewStatCache(statMaxSize, statTTL),
		fdCache:   NewFdCacheManager(fdMaxSize, fdMaxOpenFiles, pageSize),
		tree:      NewCacheTree(statMaxSize),
	}
}

// GetStatCache returns the stat cache
func (m *Manager) GetStatCache() *StatCache {
	return m.statCache
}

// GetFdCache returns the FD cache manager
func (m *Manager) GetFdCache() *FdCacheManager {
	return m.fdCache
}

// GetTree returns the cache tree
func (m *Manager) GetTree() *CacheTree {
	return m.tree
}

// Close closes all caches
func (m *Manager) Close() {
	if m.statCache != nil {
		m.statCache.Close()
	}
	if m.fdCache != nil {
		m.fdCache.CloseAll()
	}
}

// DefaultManager creates a manager with default settings
func DefaultManager() *Manager {
	return NewManager(
		1000,                    // Stat cache max size
		5*time.Minute,           // Stat cache TTL
		100,                     // FD cache max size
		10,                      // Max open files
		4096,                    // Page size
	)
}
