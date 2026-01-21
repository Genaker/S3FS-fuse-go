package cache

import (
	"sync"
	"time"
)

// CacheNode represents a node in the cache tree structure
type CacheNode struct {
	mu            sync.RWMutex
	path          string
	children      map[string]*CacheNode
	entry         *StatCacheEntry
	lastAccess    time.Time
}

// CacheTree manages a tree structure of cache nodes
type CacheTree struct {
	mu     sync.RWMutex
	root   *CacheNode
	maxSize int
}

// NewCacheTree creates a new cache tree
func NewCacheTree(maxSize int) *CacheTree {
	return &CacheTree{
		root: &CacheNode{
			path:     "",
			children: make(map[string]*CacheNode),
		},
		maxSize: maxSize,
	}
}

// Get retrieves a node from the tree
func (ct *CacheTree) Get(path string) (*CacheNode, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	parts := splitPath(path)
	node := ct.root

	for _, part := range parts {
		node.mu.RLock()
		child, exists := node.children[part]
		node.mu.RUnlock()

		if !exists {
			return nil, false
		}
		node = child
	}

	node.mu.RLock()
	defer node.mu.RUnlock()
	node.lastAccess = time.Now()
	return node, true
}

// Set stores a node in the tree
func (ct *CacheTree) Set(path string, entry *StatCacheEntry) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	parts := splitPath(path)
	node := ct.root

	for _, part := range parts {
		node.mu.Lock()
		child, exists := node.children[part]
		if !exists {
			child = &CacheNode{
				path:     part,
				children: make(map[string]*CacheNode),
			}
			node.children[part] = child
		}
		node.mu.Unlock()
		node = child
	}

	node.mu.Lock()
	node.entry = entry
	node.lastAccess = time.Now()
	node.mu.Unlock()
}

// Delete removes a node from the tree
func (ct *CacheTree) Delete(path string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	node := ct.root
	pathNodes := []*CacheNode{node}

	// Navigate to the node
	for _, part := range parts {
		node.mu.RLock()
		child, exists := node.children[part]
		node.mu.RUnlock()

		if !exists {
			return // Path doesn't exist
		}
		pathNodes = append(pathNodes, child)
		node = child
	}

	// Remove the node
	targetNode := pathNodes[len(pathNodes)-1]
	targetNode.mu.Lock()
	targetNode.entry = nil
	targetNode.mu.Unlock()

	// Clean up empty parent nodes
	for i := len(pathNodes) - 2; i >= 0; i-- {
		parent := pathNodes[i]
		parent.mu.Lock()
		childName := parts[i]
		child := parent.children[childName]
		
		child.mu.RLock()
		hasChildren := len(child.children) > 0 || child.entry != nil
		child.mu.RUnlock()

		if !hasChildren {
			delete(parent.children, childName)
		}
		parent.mu.Unlock()

		if hasChildren {
			break
		}
	}
}

// GetChildren returns all children of a node
func (ct *CacheTree) GetChildren(path string) []*CacheNode {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	parts := splitPath(path)
	node := ct.root

	for _, part := range parts {
		node.mu.RLock()
		child, exists := node.children[part]
		node.mu.RUnlock()

		if !exists {
			return nil
		}
		node = child
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	children := make([]*CacheNode, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}
	return children
}

// Clear removes all nodes from the tree
func (ct *CacheTree) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.root = &CacheNode{
		path:     "",
		children: make(map[string]*CacheNode),
	}
}

// splitPath splits a path into components
func splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}

	// Remove leading slash
	if path[0] == '/' {
		path = path[1:]
	}

	if path == "" {
		return []string{}
	}

	parts := []string{}
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
