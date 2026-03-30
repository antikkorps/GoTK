package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// entry stores a cached filter result.
type entry struct {
	key   string
	value string
	prev  *entry
	next  *entry
}

// Cache is a thread-safe LRU cache for filtered output.
// It maps a content hash (derived from raw input + filter parameters) to the filtered result.
type Cache struct {
	mu      sync.Mutex
	items   map[string]*entry
	head    *entry // most recently used
	tail    *entry // least recently used
	size    int
	maxSize int
	hits    int
	misses  int
	cfgHash string // precomputed config fingerprint
}

// New creates a new LRU cache with the given maximum number of entries.
// cfgHash is a precomputed fingerprint of the config that affects filtering.
// If maxSize <= 0, caching is disabled (Get always misses, Put is a no-op).
func New(maxSize int, cfgHash string) *Cache {
	return &Cache{
		items:   make(map[string]*entry, maxSize),
		maxSize: maxSize,
		cfgHash: cfgHash,
	}
}

// Key computes a cache key from the raw input, command type, and max lines.
// The config fingerprint is included automatically.
func (c *Cache) Key(raw string, cmdType int, maxLines int) string {
	h := sha256.New()
	h.Write([]byte(c.cfgHash))
	h.Write([]byte{0})
	fmt.Fprintf(h, "%d:%d:", cmdType, maxLines)
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached result. Returns the value and true on hit, or "" and false on miss.
func (c *Cache) Get(key string) (string, bool) {
	if c.maxSize <= 0 {
		return "", false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		c.misses++
		return "", false
	}

	c.hits++
	c.moveToFront(e)
	return e.value, true
}

// Put stores a filtered result in the cache, evicting the least recently used entry if full.
func (c *Cache) Put(key, value string) {
	if c.maxSize <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if e, ok := c.items[key]; ok {
		e.value = value
		c.moveToFront(e)
		return
	}

	// Evict LRU if at capacity
	if c.size >= c.maxSize {
		c.evict()
	}

	e := &entry{key: key, value: value}
	c.items[key] = e
	c.pushFront(e)
	c.size++
}

// Stats returns cache hit and miss counts.
func (c *Cache) Stats() (hits, misses int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hits, c.misses
}

func (c *Cache) moveToFront(e *entry) {
	if c.head == e {
		return
	}
	c.unlink(e)
	c.pushFront(e)
}

func (c *Cache) pushFront(e *entry) {
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *Cache) unlink(e *entry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
}

func (c *Cache) evict() {
	if c.tail == nil {
		return
	}
	e := c.tail
	c.unlink(e)
	delete(c.items, e.key)
	c.size--
}

// ConfigHash computes a fingerprint from config fields that affect filtering.
func ConfigHash(filtersStr, rulesStr, mode string, redactSecrets bool) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%v", filtersStr, rulesStr, mode, redactSecrets)
	return hex.EncodeToString(h.Sum(nil)[:16]) // 128 bits is enough for fingerprint
}
