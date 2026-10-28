package services

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

type CacheItem struct {
    Value      interface{}
    ExpiresAt  time.Time
    CreatedAt  time.Time
    LastAccess time.Time
    Hits       int
}

type CacheService struct {
    data       map[string]CacheItem
    mu         sync.RWMutex
    defaultTTL time.Duration
    maxSize    int
    stopCleanup chan bool
}

// NewCacheService creates a new cache service
func NewCacheService(defaultTTL time.Duration, maxSize int) *CacheService {
    c := &CacheService{
        data:        make(map[string]CacheItem),
        defaultTTL:  defaultTTL,
        maxSize:     maxSize,
        stopCleanup: make(chan bool),
    }

    // Start cleanup goroutine
    go c.startCleanup(5 * time.Minute)

    return c
}

// Set stores a value in cache
func (c *CacheService) Set(key string, value interface{}, ttl ...time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Check if we need to evict items
    if len(c.data) >= c.maxSize {
        c.evictLRU()
    }

    // Determine TTL
    expiration := time.Now().Add(c.defaultTTL)
    if len(ttl) > 0 {
        expiration = time.Now().Add(ttl[0])
    }

    c.data[key] = CacheItem{
        Value:      value,
        ExpiresAt:  expiration,
        CreatedAt:  time.Now(),
        LastAccess: time.Now(),
        Hits:       0,
    }
}

// Get retrieves a value from cache
func (c *CacheService) Get(key string) (interface{}, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    item, ok := c.data[key]
    if !ok {
        return nil, false
    }

    // Check if expired
    if time.Now().After(item.ExpiresAt) {
        delete(c.data, key)
        return nil, false
    }

    // Update access stats
    item.LastAccess = time.Now()
    item.Hits++
    c.data[key] = item

    return item.Value, true
}

// GetOrSet gets a value or sets it if not exists
func (c *CacheService) GetOrSet(key string, fetchFunc func() (interface{}, error), ttl ...time.Duration) (interface{}, error) {
    // Try to get from cache
    if val, ok := c.Get(key); ok {
        return val, nil
    }

    // Fetch value
    val, err := fetchFunc()
    if err != nil {
        return nil, err
    }

    // Store in cache
    c.Set(key, val, ttl...)

    return val, nil
}

// Delete removes a key from cache
func (c *CacheService) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.data, key)
}

// Clear removes all items from cache
func (c *CacheService) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data = make(map[string]CacheItem)
}

// Exists checks if a key exists and is not expired
func (c *CacheService) Exists(key string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, ok := c.data[key]
    if !ok {
        return false
    }

    if time.Now().After(item.ExpiresAt) {
        return false
    }

    return true
}

// TTL returns the time-to-live for a key
func (c *CacheService) TTL(key string) time.Duration {
    c.mu.RLock()
    defer c.mu.RUnlock()

    item, ok := c.data[key]
    if !ok {
        return -2 // Key doesn't exist
    }

    if time.Now().After(item.ExpiresAt) {
        return -1 // Expired
    }

    return time.Until(item.ExpiresAt)
}

// Increment increments a numeric value
func (c *CacheService) Increment(key string, delta int) (int, error) {
    c.mu.Lock()
    defer c.mu.Unlock()

    item, ok := c.data[key]
    if !ok {
        c.data[key] = CacheItem{
            Value:      delta,
            ExpiresAt:  time.Now().Add(c.defaultTTL),
            CreatedAt:  time.Now(),
            LastAccess: time.Now(),
        }
        return delta, nil
    }

    // Check if expired
    if time.Now().After(item.ExpiresAt) {
        delete(c.data, key)
        c.data[key] = CacheItem{
            Value:      delta,
            ExpiresAt:  time.Now().Add(c.defaultTTL),
            CreatedAt:  time.Now(),
            LastAccess: time.Now(),
        }
        return delta, nil
    }

    // Increment value
    switch v := item.Value.(type) {
    case int:
        item.Value = v + delta
    case float64:
        item.Value = int(v) + delta
    default:
        return 0, fmt.Errorf("value is not numeric")
    }

    item.LastAccess = time.Now()
    c.data[key] = item

    return item.Value.(int), nil
}

// GetStats returns cache statistics
func (c *CacheService) GetStats() map[string]interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var totalHits, totalItems int
    var oldestItem time.Time
    var hitRates []float64

    for _, item := range c.data {
        totalItems++
        totalHits += item.Hits
        
        if oldestItem.IsZero() || item.CreatedAt.Before(oldestItem) {
            oldestItem = item.CreatedAt
        }

        // Calculate hit rate for this item
        age := time.Since(item.CreatedAt).Hours()
        if age > 0 {
            hitRates = append(hitRates, float64(item.Hits)/age)
        }
    }

    // Calculate average hit rate
    var avgHitRate float64
    if len(hitRates) > 0 {
        var sum float64
        for _, rate := range hitRates {
            sum += rate
        }
        avgHitRate = sum / float64(len(hitRates))
    }

    return map[string]interface{}{
        "total_items":      totalItems,
        "total_hits":       totalHits,
        "avg_hit_rate":     avgHitRate,
        "oldest_item_age":  time.Since(oldestItem).Hours(),
        "max_size":         c.maxSize,
        "usage_percent":    float64(totalItems) / float64(c.maxSize) * 100,
        "default_ttl_hours": c.defaultTTL.Hours(),
    }
}

// GetMultiple retrieves multiple keys at once
func (c *CacheService) GetMultiple(keys []string) map[string]interface{} {
    result := make(map[string]interface{})

    for _, key := range keys {
        if val, ok := c.Get(key); ok {
            result[key] = val
        }
    }

    return result
}

// SetMultiple stores multiple key-value pairs
func (c *CacheService) SetMultiple(items map[string]interface{}, ttl ...time.Duration) {
    for key, value := range items {
        c.Set(key, value, ttl...)
    }
}

// DeleteMultiple removes multiple keys
func (c *CacheService) DeleteMultiple(keys []string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, key := range keys {
        delete(c.data, key)
    }
}

// GetKeys returns all keys matching pattern
func (c *CacheService) GetKeys(pattern string) []string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    var keys []string
    for key := range c.data {
        // Simple pattern matching (can be enhanced)
        if pattern == "*" || pattern == "" {
            keys = append(keys, key)
        } else if len(pattern) <= len(key) && key[:len(pattern)] == pattern {
            keys = append(keys, key)
        }
    }

    return keys
}

// Serialize returns the cache as JSON
func (c *CacheService) Serialize() ([]byte, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Create a serializable version
    serializable := make(map[string]interface{})
    for key, item := range c.data {
        if time.Now().Before(item.ExpiresAt) {
            serializable[key] = item.Value
        }
    }

    return json.Marshal(serializable)
}

// Deserialize loads cache from JSON
func (c *CacheService) Deserialize(data []byte) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    var items map[string]interface{}
    if err := json.Unmarshal(data, &items); err != nil {
        return err
    }

    now := time.Now()
    for key, value := range items {
        c.data[key] = CacheItem{
            Value:      value,
            ExpiresAt:  now.Add(c.defaultTTL),
            CreatedAt:  now,
            LastAccess: now,
        }
    }

    return nil
}

// Shutdown stops the cleanup goroutine
func (c *CacheService) Shutdown() {
    c.stopCleanup <- true
}

// Private methods

func (c *CacheService) startCleanup(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            c.cleanup()
        case <-c.stopCleanup:
            return
        }
    }
}

func (c *CacheService) cleanup() {
    c.mu.Lock()
    defer c.mu.Unlock()

    now := time.Now()
    for key, item := range c.data {
        if now.After(item.ExpiresAt) {
            delete(c.data, key)
        }
    }
}

func (c *CacheService) evictLRU() {
    if len(c.data) == 0 {
        return
    }

    // Find least recently used item
    var lruKey string
    var lruTime time.Time

    for key, item := range c.data {
        if lruTime.IsZero() || item.LastAccess.Before(lruTime) {
            lruKey = key
            lruTime = item.LastAccess
        }
    }

    // Evict it
    delete(c.data, lruKey)
}