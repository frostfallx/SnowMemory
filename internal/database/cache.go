package database

import (
	"sync"
	"time"
)

// cacheEntry 缓存条目
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// MemoryCache 内存缓存
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheEntry
	ttl   time.Duration
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	c := &MemoryCache{
		items: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
	// 启动定期清理
	go c.cleanupLoop()
	return c
}

// Get 获取缓存值
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// Set 设置缓存值
func (c *MemoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete 删除缓存
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// DeleteByPrefix 按前缀删除
func (c *MemoryCache) DeleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

// Clear 清空缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheEntry)
}

// cleanupLoop 定期清理过期条目
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// 全局缓存实例
var (
	userCache  *MemoryCache
	aliasCache *MemoryCache
	factCache  *MemoryCache
	cacheOnce  sync.Once
)

// InitCache 初始化缓存层
func InitCache(ttl time.Duration) {
	cacheOnce.Do(func() {
		userCache = NewMemoryCache(ttl)
		aliasCache = NewMemoryCache(ttl)
		factCache = NewMemoryCache(ttl)
	})
}

// InvalidateUserCache 失效用户缓存
func InvalidateUserCache(userID string) {
	if userCache != nil {
		userCache.Delete("user:" + userID)
		userCache.Delete("user_profile:" + userID)
	}
}

// InvalidateAliasCache 失效别名缓存
func InvalidateAliasCache(userID string) {
	if aliasCache != nil {
		aliasCache.DeleteByPrefix("aliases:" + userID)
	}
}

// InvalidateFactCache 失效事实缓存
func InvalidateFactCache(userID string) {
	if factCache != nil {
		factCache.DeleteByPrefix("facts:" + userID)
	}
}
