// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package lrucache provides a thread-safe, in-memory LRU (Least Recently Used) cache
// with TTL (Time To Live) support and size-based eviction.
//
// This cache is designed for multi-threaded applications where expensive computations
// or I/O operations need to be cached. It provides automatic synchronization to ensure
// that the same value is not computed multiple times concurrently.
//
// Key features:
//   - Thread-safe: Safe for concurrent access from multiple goroutines
//   - LRU eviction: Automatically evicts least recently used entries when memory limit is reached
//   - TTL support: Entries expire after a configurable time-to-live
//   - Lazy computation: Values are computed on-demand and only once per key
//   - Concurrent computation prevention: If multiple goroutines request the same key,
//     only one computes the value while others wait for the result
//   - HTTP middleware: Includes an HTTP handler for caching HTTP responses
//
// Basic usage:
//
//	cache := lrucache.New(1000) // maxmemory in arbitrary units
//
//	value := cache.Get("key", func() (interface{}, time.Duration, int) {
//	    // This closure is called only if the value is not cached or expired
//	    result := expensiveComputation()
//	    return result, 10 * time.Minute, len(result) // value, ttl, size
//	})
//
// The size parameter is a user-defined estimate in any consistent unit. It can be:
//   - Actual bytes: len(string) or len(slice) * sizeof(element)
//   - Entry count: Use 1 for each entry to limit by number of entries
//   - Custom metric: Any measure that makes sense for your use case
//
// See the README.md for more detailed examples and explanations.
package lrucache

import (
	"sync"
	"time"
)

// ComputeValue is the type of the closure that must be passed to Get to
// compute a value when it is not cached or has expired.
//
// The closure should perform the expensive computation or I/O operation
// and return three values:
//
//   - value: The computed value to be stored in the cache (can be any type)
//   - ttl: Time-to-live duration until this value expires and needs recomputation
//   - size: A size estimate in user-defined units (see package documentation)
//
// The closure is called synchronously and must not call methods on the same
// cache instance to avoid deadlocks. If multiple goroutines request the same
// key concurrently, only one will execute this closure while others wait.
//
// Example:
//
//	computeValue := func() (interface{}, time.Duration, int) {
//	    data := fetchFromDatabase() // Expensive operation
//	    return data, 5 * time.Minute, len(data)
//	}
type ComputeValue func() (value any, ttl time.Duration, size int)

// cacheEntry represents a single entry in the LRU cache.
// It is part of a doubly-linked list for LRU tracking.
type cacheEntry struct {
	key   string // Cache key
	value any    // Cached value

	// expiration is the time when this entry expires.
	// A zero value indicates the value is currently being computed.
	expiration time.Time

	// size is the user-provided size estimate for this entry
	size int

	// waitingForComputation tracks how many goroutines are waiting
	// for this value to be computed
	waitingForComputation int

	// Doubly-linked list pointers for LRU ordering
	// (most recently used at head, least recently used at tail)
	next, prev *cacheEntry
}

// Cache is a thread-safe LRU cache with TTL support.
//
// The cache uses a mutex for synchronization and a condition variable
// to coordinate goroutines waiting for values being computed.
//
// Concurrency model:
//   - All public methods are thread-safe
//   - Multiple goroutines can read different keys concurrently
//   - If multiple goroutines request the same uncached key, only one
//     computes the value while others wait
//   - The cache uses a condition variable (cond) to wake up waiting goroutines
//
// Memory management:
//   - maxmemory: Maximum total size (in user-defined units)
//   - usedmemory: Current total size of all cached entries
//   - When usedmemory exceeds maxmemory, least recently used entries are evicted
//
// Data structures:
//   - entries: Hash map for O(1) key lookup
//   - head/tail: Doubly-linked list for LRU ordering (head = most recent)
type Cache struct {
	mutex                 sync.Mutex             // Protects all cache operations
	cond                  *sync.Cond             // Coordinates waiting goroutines
	maxmemory, usedmemory int                    // Memory limits and usage
	entries               map[string]*cacheEntry // Fast key lookup
	head, tail            *cacheEntry            // LRU list (head=newest, tail=oldest)
}

// New creates and returns a new LRU cache instance.
//
// The maxmemory parameter sets the maximum total size of all cached entries.
// The size is measured in user-defined units (see package documentation).
// When the total size exceeds maxmemory, the least recently used entries
// are evicted until the size is below the limit.
//
// Common strategies for maxmemory:
//   - Bytes: Set to actual memory limit (e.g., 100*1024*1024 for 100MB)
//   - Entry count: Set to max number of entries (use size=1 for each entry)
//   - Custom: Any consistent unit that makes sense for your use case
//
// Example:
//
//	// Limit cache to approximately 10MB
//	cache := lrucache.New(10 * 1024 * 1024)
//
//	// Limit cache to 1000 entries (using size=1 per entry)
//	cache := lrucache.New(1000)
func New(maxmemory int) *Cache {
	cache := &Cache{
		maxmemory: maxmemory,
		entries:   map[string]*cacheEntry{},
	}
	cache.cond = sync.NewCond(&cache.mutex)
	return cache
}

// Get retrieves the cached value for the given key or computes it using computeValue.
//
// Behavior:
//   - If the key exists and hasn't expired: Returns the cached value immediately
//   - If the key doesn't exist or has expired: Calls computeValue to compute the value
//   - If computeValue is nil and key not found: Returns nil
//   - If another goroutine is computing the same key: Waits for that computation to complete
//
// Concurrency guarantees:
//   - Only one goroutine will execute computeValue for a given key at a time
//   - Other goroutines requesting the same key will wait for the result
//   - Different keys can be computed concurrently without blocking each other
//   - The computeValue closure is called synchronously (not in a separate goroutine)
//
// IMPORTANT: The computeValue closure must NOT call methods on the same cache
// instance, as this will cause a deadlock. If you need to access other cache
// entries, do so before or after the Get call.
//
// Parameters:
//   - key: The cache key to look up
//   - computeValue: Closure to compute the value if not cached (can be nil for lookup-only)
//
// Returns:
//   - The cached or computed value, or nil if computeValue is nil and key not found
//
// Examples:
//
//	// Basic usage with computation
//	value := cache.Get("user:123", func() (interface{}, time.Duration, int) {
//	    user := fetchUserFromDB(123)
//	    return user, 10 * time.Minute, 1
//	}).(User)
//
//	// Lookup-only (no computation)
//	value := cache.Get("user:123", nil)
//	if value == nil {
//	    // Key not found or expired
//	}
//
//	// With size calculation
//	value := cache.Get("data", func() (interface{}, time.Duration, int) {
//	    data := expensiveComputation()
//	    return data, 1 * time.Hour, len(data) * 8 // Approximate bytes
//	})
func (c *Cache) Get(key string, computeValue ComputeValue) any {
	now := time.Now()

	c.mutex.Lock()
	if entry, ok := c.entries[key]; ok {
		// The expiration not being set is what shows us that
		// the computation of that value is still ongoing.
		for entry.expiration.IsZero() {
			entry.waitingForComputation += 1
			c.cond.Wait()
			entry.waitingForComputation -= 1
		}

		if now.After(entry.expiration) {
			if !c.evictEntry(entry) {
				if entry.expiration.IsZero() {
					panic("LRUCACHE/CACHE > cache entry that shoud have been waited for could not be evicted.")
				}
				c.mutex.Unlock()
				return entry.value
			}
		} else {
			if entry != c.head {
				c.unlinkEntry(entry)
				c.insertFront(entry)
			}
			c.mutex.Unlock()
			return entry.value
		}
	}

	if computeValue == nil {
		c.mutex.Unlock()
		return nil
	}

	entry := &cacheEntry{
		key:                   key,
		waitingForComputation: 1,
	}

	c.entries[key] = entry

	hasPaniced := true
	defer func() {
		if hasPaniced {
			c.mutex.Lock()
			delete(c.entries, key)
			entry.expiration = now
			entry.waitingForComputation -= 1
		}
		c.mutex.Unlock()
	}()

	c.mutex.Unlock()
	value, ttl, size := computeValue()
	c.mutex.Lock()
	hasPaniced = false

	entry.value = value
	entry.expiration = now.Add(ttl)
	entry.size = size
	entry.waitingForComputation -= 1

	// Only broadcast if other goroutines are actually waiting
	// for a result.
	if entry.waitingForComputation > 0 {
		// TODO: Have more than one condition variable so that there are
		// less unnecessary wakeups.
		c.cond.Broadcast()
	}

	c.usedmemory += size
	c.insertFront(entry)

	// Evict only entries with a size of more than zero.
	// This is the only loop in the implementation outside of the `Keys`
	// method.
	evictionCandidate := c.tail
	for c.usedmemory > c.maxmemory && evictionCandidate != nil {
		nextCandidate := evictionCandidate.prev
		if (evictionCandidate.size > 0 || now.After(evictionCandidate.expiration)) &&
			evictionCandidate.waitingForComputation == 0 {
			c.evictEntry(evictionCandidate)
		}
		evictionCandidate = nextCandidate
	}

	return value
}

// Put stores a value in the cache with the specified key, size, and TTL.
//
// If another goroutine is currently computing this key via Get, Put will
// wait for the computation to complete before overwriting the value.
//
// If the key already exists, the old value is replaced and the entry is
// moved to the front of the LRU list (marked as most recently used).
//
// Parameters:
//   - key: The cache key
//   - value: The value to store (can be any type)
//   - size: Size estimate in user-defined units
//   - ttl: Time-to-live duration until the value expires
//
// Example:
//
//	cache.Put("config", configData, len(configData), 1 * time.Hour)
func (c *Cache) Put(key string, value any, size int, ttl time.Duration) {
	now := time.Now()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, ok := c.entries[key]; ok {
		for entry.expiration.IsZero() {
			entry.waitingForComputation += 1
			c.cond.Wait()
			entry.waitingForComputation -= 1
		}

		c.usedmemory -= entry.size
		entry.expiration = now.Add(ttl)
		entry.size = size
		entry.value = value
		c.usedmemory += entry.size

		c.unlinkEntry(entry)
		c.insertFront(entry)
		return
	}

	entry := &cacheEntry{
		key:        key,
		value:      value,
		expiration: now.Add(ttl),
	}
	c.entries[key] = entry
	c.insertFront(entry)
}

// Del removes the entry with the given key from the cache.
//
// Returns:
//   - true if the key was in the cache (even if expired)
//   - false if the key was not found
//
// Note: This function may return false even if the value will appear in the
// cache later, if called while another goroutine is computing that key.
// It may return true even if the value has already expired.
//
// Example:
//
//	if cache.Del("old-key") {
//	    log.Println("Removed old-key from cache")
//	}
func (c *Cache) Del(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, ok := c.entries[key]; ok {
		return c.evictEntry(entry)
	}
	return false
}

// Keys iterates over all entries in the cache and calls f for each one.
//
// The function f receives the key and value of each entry. During iteration,
// expired entries are automatically evicted and sanity checks are performed
// on the internal data structures.
//
// IMPORTANT: The cache is fully locked for the entire duration of this call.
// This means no other cache operations can proceed while Keys is running.
// Keep the function f as fast as possible to minimize lock contention.
//
// The iteration order is not guaranteed.
//
// Example:
//
//	cache.Keys(func(key string, val interface{}) {
//	    fmt.Printf("Key: %s, Value: %v\n", key, val)
//	})
func (c *Cache) Keys(f func(key string, val any)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	size := 0
	for key, e := range c.entries {
		if key != e.key {
			panic("LRUCACHE/CACHE > key mismatch")
		}

		if now.After(e.expiration) {
			if c.evictEntry(e) {
				continue
			}
		}

		if e.prev != nil {
			if e.prev.next != e {
				panic("LRUCACHE/CACHE > list corrupted")
			}
		}

		if e.next != nil {
			if e.next.prev != e {
				panic("LRUCACHE/CACHE > list corrupted")
			}
		}

		size += e.size
		f(key, e.value)
	}

	if size != c.usedmemory {
		panic("LRUCACHE/CACHE > size calculations failed")
	}

	if c.head != nil {
		if c.tail == nil || c.head.prev != nil {
			panic("LRUCACHE/CACHE > head/tail corrupted")
		}
	}

	if c.tail != nil {
		if c.head == nil || c.tail.next != nil {
			panic("LRUCACHE/CACHE > head/tail corrupted")
		}
	}
}

// insertFront adds an entry to the front of the LRU list (most recently used position).
func (c *Cache) insertFront(e *cacheEntry) {
	e.next = c.head
	c.head = e

	e.prev = nil
	if e.next != nil {
		e.next.prev = e
	}

	if c.tail == nil {
		c.tail = e
	}
}

// unlinkEntry removes an entry from the doubly-linked list without deleting it from the map.
func (c *Cache) unlinkEntry(e *cacheEntry) {
	if e == c.head {
		c.head = e.next
	}
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == c.tail {
		c.tail = e.prev
	}
}

// evictEntry removes an entry from both the list and the map.
// Returns false if the entry cannot be evicted (other goroutines are waiting for it).
func (c *Cache) evictEntry(e *cacheEntry) bool {
	if e.waitingForComputation != 0 {
		// panic("LRUCACHE/CACHE > cannot evict this entry as other goroutines need the value")
		return false
	}

	c.unlinkEntry(e)
	c.usedmemory -= e.size
	delete(c.entries, e.key)
	return true
}
