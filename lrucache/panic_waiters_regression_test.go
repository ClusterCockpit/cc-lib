// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package lrucache

import (
	"runtime"
	"testing"
	"time"
)

func waitForComputationWaiters(t *testing.T, cache *Cache, key string, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		cache.mutex.Lock()
		entry := cache.entries[key]
		ready := entry != nil && entry.expiration.IsZero() && entry.waitingForComputation == count
		cache.mutex.Unlock()
		if ready {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("goroutines did not reach the observed cache wait state")
}

func TestGetPanicUnblocksGetAndPut(t *testing.T) {
	cache := New(100)
	cache.Put("unrelated", "preserved", 1, time.Hour)
	started := make(chan struct{})
	panicNow := make(chan struct{})
	panicDone := make(chan any, 1)
	go func() {
		defer func() { panicDone <- recover() }()
		cache.Get("key", func() (any, time.Duration, int) {
			close(started)
			<-panicNow
			panic("expected compute panic")
		})
	}()
	<-started
	getDone := make(chan any, 1)
	putDone := make(chan struct{})
	go func() {
		getDone <- cache.Get("key", func() (any, time.Duration, int) {
			return "recomputed", time.Hour, 2
		})
	}()
	go func() {
		cache.Put("key", "put", 3, time.Hour)
		close(putDone)
	}()
	waitForComputationWaiters(t, cache, "key", 3)
	close(panicNow)
	if value := <-panicDone; value != "expected compute panic" {
		t.Fatalf("compute panic was not preserved: %v", value)
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	var got any
	select {
	case got = <-getDone:
	case <-timer.C:
		t.Fatal("Get remained blocked after ComputeValue panicked")
	}
	select {
	case <-putDone:
	case <-timer.C:
		t.Fatal("Put remained blocked after ComputeValue panicked")
	}
	if got != "recomputed" && got != "put" {
		t.Fatalf("waiting Get lost its value: %v", got)
	}
	if cache.Get("key", nil) != "put" || cache.Get("unrelated", nil) != "preserved" {
		t.Fatal("recovery lost the new value or an unrelated entry")
	}
	cache.Keys(func(string, any) {}) // Checks map/list and memory accounting.
	if cache.usedmemory != 4 {
		t.Fatalf("unexpected recovered size: %d", cache.usedmemory)
	}
}

func TestGetPanicUnblocksLookupOnly(t *testing.T) {
	cache := New(100)
	started, panicNow := make(chan struct{}), make(chan struct{})
	panicDone := make(chan struct{})
	go func() {
		defer func() { _ = recover(); close(panicDone) }()
		cache.Get("key", func() (any, time.Duration, int) {
			close(started)
			<-panicNow
			panic("expected")
		})
	}()
	<-started
	lookupDone := make(chan any, 1)
	go func() { lookupDone <- cache.Get("key", nil) }()
	waitForComputationWaiters(t, cache, "key", 2)
	close(panicNow)
	<-panicDone
	select {
	case value := <-lookupDone:
		if value != nil {
			t.Fatalf("lookup-only Get returned an abandoned value: %v", value)
		}
	case <-time.After(time.Second):
		t.Fatal("lookup-only Get remained blocked after ComputeValue panicked")
	}
}
