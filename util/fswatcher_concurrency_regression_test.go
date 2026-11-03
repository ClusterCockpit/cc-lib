// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"sync"
	"testing"
)

type auditWatcherListener struct{}

func (*auditWatcherListener) EventMatch(string) bool { return false }
func (*auditWatcherListener) EventCallback()         {}
func TestFsWatcherConcurrentListenerRegistration(t *testing.T) {
	directory := t.TempDir()
	AddListener(directory, &auditWatcherListener{})
	defer FsWatcherShutdown()
	start := make(chan struct{})
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() { defer group.Done(); <-start; AddListener(directory, &auditWatcherListener{}) }()
	}
	close(start)
	group.Wait()
	registered := len(listeners)
	if registered != 33 {
		t.Fatalf("registered %d listeners, want 33", registered)
	}
}
