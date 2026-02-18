// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package lrucache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestBasics validates basic cache operations: Get, Put, Del, and Keys.
// It tests that values are cached correctly, retrieved without recomputation,
// and can be deleted and recomputed.
func TestBasics(t *testing.T) {
	cache := New(123)

	value1 := cache.Get("foo", func() (any, time.Duration, int) {
		return "bar", 1 * time.Second, 0
	})

	if value1.(string) != "bar" {
		t.Error("cache returned wrong value")
	}

	value2 := cache.Get("foo", func() (any, time.Duration, int) {
		t.Error("value should be cached")
		return "", 0, 0
	})

	if value2.(string) != "bar" {
		t.Error("cache returned wrong value")
	}

	existed := cache.Del("foo")
	if !existed {
		t.Error("delete did not work as expected")
	}

	value3 := cache.Get("foo", func() (any, time.Duration, int) {
		return "baz", 1 * time.Second, 0
	})

	if value3.(string) != "baz" {
		t.Error("cache returned wrong value")
	}

	cache.Keys(func(key string, value any) {
		if key != "foo" || value.(string) != "baz" {
			t.Error("cache corrupted")
		}
	})
}

// TestExpiration validates that cache entries expire correctly based on their TTL.
// It tests that expired entries are recomputed while non-expired entries are served
// from cache, and that the Keys function properly evicts expired entries.
func TestExpiration(t *testing.T) {
	cache := New(123)

	failIfCalled := func() (any, time.Duration, int) {
		t.Error("Value should be cached!")
		return "", 0, 0
	}

	val1 := cache.Get("foo", func() (any, time.Duration, int) {
		return "bar", 5 * time.Millisecond, 0
	})
	val2 := cache.Get("bar", func() (any, time.Duration, int) {
		return "foo", 20 * time.Millisecond, 0
	})

	val3 := cache.Get("foo", failIfCalled).(string)
	val4 := cache.Get("bar", failIfCalled).(string)

	if val1 != val3 || val3 != "bar" || val2 != val4 || val4 != "foo" {
		t.Error("Wrong values returned")
	}

	time.Sleep(10 * time.Millisecond)

	val5 := cache.Get("foo", func() (any, time.Duration, int) {
		return "baz", 0, 0
	})
	val6 := cache.Get("bar", failIfCalled)

	if val5.(string) != "baz" || val6.(string) != "foo" {
		t.Error("unexpected values")
	}

	cache.Keys(func(key string, val any) {
		if key != "bar" || val.(string) != "foo" {
			t.Error("wrong value expired")
		}
	})

	time.Sleep(15 * time.Millisecond)
	cache.Keys(func(key string, val any) {
		t.Error("cache should be empty now")
	})
}

// TestEviction validates the LRU eviction policy.
// It tests that entries are evicted when the cache size limit is exceeded,
// and that the least recently used entries are evicted first.
func TestEviction(t *testing.T) {
	c := New(100)
	failIfCalled := func() (any, time.Duration, int) {
		t.Error("Value should be cached!")
		return "", 0, 0
	}

	v1 := c.Get("foo", func() (any, time.Duration, int) {
		return "bar", 1 * time.Second, 1000
	})

	v2 := c.Get("foo", func() (any, time.Duration, int) {
		return "baz", 1 * time.Second, 1000
	})

	if v1.(string) != "bar" || v2.(string) != "baz" {
		t.Error("wrong values returned")
	}

	c.Keys(func(key string, val any) {
		t.Error("cache should be empty now")
	})

	_ = c.Get("A", func() (any, time.Duration, int) {
		return "a", 1 * time.Second, 50
	})

	_ = c.Get("B", func() (any, time.Duration, int) {
		return "b", 1 * time.Second, 50
	})

	_ = c.Get("A", failIfCalled)
	_ = c.Get("B", failIfCalled)
	_ = c.Get("C", func() (any, time.Duration, int) {
		return "c", 1 * time.Second, 50
	})

	_ = c.Get("B", failIfCalled)
	_ = c.Get("C", failIfCalled)

	v4 := c.Get("A", func() (any, time.Duration, int) {
		return "evicted", 1 * time.Second, 25
	})

	if v4.(string) != "evicted" {
		t.Error("value should have been evicted")
	}

	c.Keys(func(key string, val any) {
		if key != "A" && key != "C" {
			t.Errorf("'%s' was not expected", key)
		}
	})
}

// TestConcurrency validates that concurrent access to the same key is handled correctly.
// It ensures that only one goroutine computes a value for a given key at a time,
// while other goroutines wait for the result.
//
// Note: This test uses timing assumptions and may be flaky on heavily loaded systems.
func TestConcurrency(t *testing.T) {
	c := New(100)
	var wg sync.WaitGroup

	numActions := 20000
	numThreads := 4
	wg.Add(numThreads)

	var concurrentModifications int32 = 0

	for range numThreads {
		go func() {
			for range numActions {
				_ = c.Get("key", func() (any, time.Duration, int) {
					m := atomic.AddInt32(&concurrentModifications, 1)
					if m != 1 {
						t.Error("only one goroutine at a time should calculate a value for the same key")
					}

					time.Sleep(1 * time.Millisecond)
					atomic.AddInt32(&concurrentModifications, -1)
					return "value", 3 * time.Millisecond, 1
				})
			}

			wg.Done()
		}()
	}

	wg.Wait()

	c.Keys(func(key string, val any) {})
}

// TestPanic validates that panics in the compute function are handled correctly.
// It ensures that the cache remains in a consistent state after a panic,
// and that other keys can still be accessed normally.
func TestPanic(t *testing.T) {
	c := New(100)

	c.Put("bar", "baz", 3, 1*time.Minute)

	testpanic := func() {
		defer func() {
			if r := recover(); r != nil {
				if r.(string) != "oops" {
					t.Fatal("unexpected panic value")
				}
			}
		}()

		_ = c.Get("foo", func() (value any, ttl time.Duration, size int) {
			panic("oops")
		})

		t.Fatal("should have paniced!")
	}

	testpanic()

	v := c.Get("bar", func() (value any, ttl time.Duration, size int) {
		t.Fatal("should not be called!")
		return nil, 0, 0
	})

	if v.(string) != "baz" {
		t.Fatal("unexpected value")
	}

	testpanic()
}
