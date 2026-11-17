// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package lrucache

import (
	"testing"
	"time"
)

func TestPutChargesNewEntriesAndEvictsLeastRecent(t *testing.T) {
	cache := New(10)
	cache.Put("first", 1, 6, time.Hour)
	cache.Put("second", 2, 6, time.Hour)
	if cache.Get("first", nil) != nil || cache.Get("second", nil) != 2 || cache.usedmemory != 6 {
		t.Fatalf("new Put entries violated budget: %d charged, %d entries", cache.usedmemory, len(cache.entries))
	}
}

func TestPutUpdateAlsoEnforcesBudget(t *testing.T) {
	cache := New(10)
	cache.Get("first", func() (any, time.Duration, int) { return 1, time.Hour, 4 })
	cache.Get("second", func() (any, time.Duration, int) { return 2, time.Hour, 4 })
	cache.Put("second", 3, 9, time.Hour)
	if cache.Get("first", nil) != nil || cache.Get("second", nil) != 3 || cache.usedmemory != 9 {
		t.Fatalf("Put update violated budget: %d charged, %d entries", cache.usedmemory, len(cache.entries))
	}
}

func TestPutRetainsZeroCostEntriesAndDropsOversizedEntries(t *testing.T) {
	cache := New(10)
	cache.Put("zero", 0, 0, time.Hour)
	cache.Put("oversized", 1, 11, time.Hour)
	if cache.Get("zero", nil) != 0 || cache.Get("oversized", nil) != nil || cache.usedmemory != 0 {
		t.Fatal("zero-cost or oversized entry behavior is incorrect")
	}
}
