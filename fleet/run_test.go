// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBootstrap_Online(t *testing.T) {
	b := newFakeBackend(t)
	b.setConfig(`{"interval":"5s"}`, "1")
	c := newTestClient(t, b, nil)

	u, err := c.Bootstrap(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if u.Source != SourceFleet || string(u.Config) != `{"interval":"5s"}` {
		t.Errorf("got %+v", u)
	}
}

func TestBootstrap_NoConfigIsNotAnError(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	u, err := c.Bootstrap(context.Background(), time.Second)
	if err != nil || u.Source != SourceNone || u.Config != nil {
		t.Errorf("got (%+v, %v), want SourceNone without error", u, err)
	}
}

func TestBootstrap_OfflineUsesCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "sub", "fleet.json")
	b := newFakeBackend(t)
	b.setConfig(`{"interval":"5s"}`, "1")

	online := newTestClient(t, b, func(o *Options) { o.Config.CachePath = cachePath })
	if _, err := online.Bootstrap(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("cache mode = %v, want 0600", st.Mode().Perm())
	}
	if data, _ := os.ReadFile(cachePath); bytes.Contains(data, []byte(online.id())) {
		t.Error("cache file must not contain the instance id")
	}

	url := b.srv.URL
	b.srv.Close()
	offline := newTestClient(t, b, func(o *Options) {
		o.Config.CachePath = cachePath
		o.Config.URL = url
	})
	u, err := offline.Bootstrap(context.Background(), time.Second)
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if u.Source != SourceCache || string(u.Config) != `{"interval":"5s"}` || u.Revision != "1" {
		t.Errorf("got %+v, want cached config", u)
	}
}

func TestBootstrap_CacheGives304(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "fleet.json")
	b := newFakeBackend(t)
	b.setConfig(`{"a":1}`, "1")
	first := newTestClient(t, b, func(o *Options) { o.Config.CachePath = cachePath })
	if _, err := first.Bootstrap(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}

	second := newTestClient(t, b, func(o *Options) { o.Config.CachePath = cachePath })
	u, err := second.Bootstrap(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if u.Source != SourceFleet || string(u.Config) != `{"a":1}` {
		t.Errorf("got %+v", u)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lastIfNoneM != `"1"` {
		t.Errorf("If-None-Match = %q, want the cached ETag", b.lastIfNoneM)
	}
}

func TestBootstrap_CacheIdentityMismatch(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "fleet.json")
	b := newFakeBackend(t)
	b.setConfig(`{"a":1}`, "1")
	first := newTestClient(t, b, func(o *Options) { o.Config.CachePath = cachePath })
	if _, err := first.Bootstrap(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}

	url := b.srv.URL
	b.srv.Close()
	other := newTestClient(t, b, func(o *Options) {
		o.Config.CachePath = cachePath
		o.Config.URL = url
		o.Config.Cluster = "alex"
	})
	u, err := other.Bootstrap(context.Background(), time.Second)
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if u.Source != SourceNone || u.Config != nil {
		t.Errorf("got %+v, want the cache of another cluster to be ignored", u)
	}
}

func TestBootstrap_UnauthorizedIsNotUnreachable(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, func(o *Options) { o.Config.Token = "wrong" })
	_, err := c.Bootstrap(context.Background(), time.Second)
	if !errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrUnreachable) {
		t.Errorf("err = %v, want ErrUnauthorized only", err)
	}
}

func TestRun_ReregistersAndDelivers(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := c.Bootstrap(ctx, time.Second); err != nil {
		t.Fatal(err)
	}
	b.rotate()
	b.setConfig(`{"a":1}`, "1")

	runErr := make(chan error, 1)
	go func() { runErr <- c.Run(ctx) }()

	select {
	case u := <-c.Configs():
		if string(u.Config) != `{"a":1}` {
			t.Errorf("config = %s", u.Config)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no config delivered")
	}
	if n := b.get(func(b *fakeBackend) int { return b.registrations }); n < 2 {
		t.Errorf("registrations = %d, want re-registration", n)
	}
	waitFor(t, "REST heartbeats", func() bool {
		return b.get(func(b *fakeBackend) int { return b.restBeats }) > 0
	})

	if err := c.Run(ctx); err == nil {
		t.Error("second Run should fail")
	}
	cancel()
	if err := <-runErr; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

func TestRun_BacksOffWhenUnauthorized(t *testing.T) {
	b := newFakeBackend(t)
	b.registerStatus = http.StatusForbidden
	c := newTestClient(t, b, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_ = c.Run(ctx)

	// 20ms ticks over 300ms would give ~30 attempts without backoff;
	// the 1s minimum backoff allows exactly one.
	if n := b.get(func(b *fakeBackend) int { return b.registerAttempts }); n != 1 {
		t.Errorf("register attempts = %d, want 1 within the first backoff", n)
	}
}

func TestClose_WaitsForRunThenDeregisters(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)

	runErr := make(chan error, 1)
	go func() { runErr <- c.Run(context.Background()) }()
	waitFor(t, "registration", c.Registered)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Close(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run returned %v", err)
		}
	default:
		t.Fatal("Close returned before Run")
	}
	if n := b.get(func(b *fakeBackend) int { return b.deregistered }); n != 1 {
		t.Errorf("deregistrations = %d, want 1", n)
	}
	if c.Registered() {
		t.Error("still registered after Close")
	}
	if err := c.Run(context.Background()); err == nil {
		t.Error("Run after Close should fail")
	}
}

func TestOfferLatest_KeepsNewest(t *testing.T) {
	ch := make(chan int, 1)
	for i := 1; i <= 5; i++ {
		offerLatest(ch, i)
	}
	if got := <-ch; got != 5 {
		t.Errorf("got %d, want 5", got)
	}
	select {
	case v := <-ch:
		t.Errorf("unexpected extra value %d", v)
	default:
	}
}

func TestRetryState_Backoff(t *testing.T) {
	var r retryState
	now := time.Now()
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for _, d := range want {
		r.fail(now)
		if got := r.next.Sub(now); got != d {
			t.Errorf("backoff = %v, want %v", got, d)
		}
	}
	for range 30 {
		r.fail(now)
	}
	if got := r.next.Sub(now); got != maxBackoff {
		t.Errorf("backoff = %v, want cap %v", got, maxBackoff)
	}
}
