package tool

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
)

// CachingRegistry wraps a Registry and caches tool call results within a scan.
// Same tool + same params returns the cached result instead of re-executing.
type CachingRegistry struct {
	inner *Registry
	mu    sync.RWMutex
	cache map[string]string
}

// NewCachingRegistry creates a caching wrapper around a Registry.
func NewCachingRegistry(inner *Registry) *CachingRegistry {
	return &CachingRegistry{
		inner: inner,
		cache: make(map[string]string),
	}
}

func cacheKey(name string, params json.RawMessage) string {
	h := sha256.Sum256(append([]byte(name+"|"), params...))
	return fmt.Sprintf("%x", h[:16])
}

// Call executes a tool, returning a cached result if available.
func (c *CachingRegistry) Call(ctx context.Context, name string, params json.RawMessage) (string, error) {
	key := cacheKey(name, params)

	c.mu.RLock()
	if result, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	result, err := c.inner.Call(ctx, name, params)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.cache[key] = result
	c.mu.Unlock()

	return result, nil
}

// Definitions delegates to the inner registry.
func (c *CachingRegistry) Definitions() []Definition {
	return c.inner.Definitions()
}

// Hashes returns a map of cache keys to content hashes for change detection.
func (c *CachingRegistry) Hashes() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hashes := make(map[string]string, len(c.cache))
	for k, v := range c.cache {
		h := sha256.Sum256([]byte(v))
		hashes[k] = fmt.Sprintf("%x", h[:16])
	}
	return hashes
}
