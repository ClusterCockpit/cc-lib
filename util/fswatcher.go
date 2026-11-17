// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package util provides common utility functions and types for ClusterCockpit applications.
// This file implements a file system watcher for monitoring file changes.
package util

import (
	"sync"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"github.com/fsnotify/fsnotify"
)

// Listener is an interface for file system event callbacks.
// Implementations should define what events to match and how to respond to them.
type Listener interface {
	EventCallback()
	EventMatch(event string) bool
}

var (
	listenersMu sync.RWMutex
	initOnce    sync.Once
	w           *fsnotify.Watcher
	listeners   []Listener
)

// AddListener registers a new file system watcher for the specified path.
// The watcher is initialized on the first call to AddListener.
// The listener will be notified of file system events matching its EventMatch criteria.
func AddListener(path string, l Listener) {
	initOnce.Do(func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			cclog.Errorf("creating a new watcher: %v", err)
			return
		}
		listenersMu.Lock()
		w = watcher
		listeners = make([]Listener, 0)
		listenersMu.Unlock()
		go watchLoop(watcher)
	})
	listenersMu.Lock()
	watcher := w
	if watcher == nil {
		listenersMu.Unlock()
		return
	}
	listeners = append(listeners, l)
	listenersMu.Unlock()
	if err := watcher.Add(path); err != nil {
		cclog.Warnf("%q: %s", path, err)
	}
}

// FsWatcherShutdown closes the file system watcher.
// This should be called during application shutdown to clean up resources.
func FsWatcherShutdown() {
	listenersMu.RLock()
	watcher := w
	listenersMu.RUnlock()
	if watcher != nil {
		watcher.Close()
	}
}

func watchLoop(w *fsnotify.Watcher) {
	for {
		select {
		// Read from Errors.
		case err, ok := <-w.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			cclog.Errorf("watch event loop: %s", err)
		// Read from Events.
		case e, ok := <-w.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			cclog.Infof("Event %s", e)
			listenersMu.RLock()
			current := append([]Listener(nil), listeners...)
			listenersMu.RUnlock()
			for _, l := range current {
				if l.EventMatch(e.String()) {
					l.EventCallback()
				}
			}
		}
	}
}
