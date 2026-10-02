// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const testToken = "secret-jwt"

// fakeBackend imitates the cc-backend fleet REST API.
type fakeBackend struct {
	t   *testing.T
	srv *httptest.Server

	mu               sync.Mutex
	nextID           int
	current          string // valid instance id; empty after rotate/deregister
	registerStatus   int    // 0: 201
	registerAttempts int
	lastRegPath      string
	lastReg          RegisterRequest
	registrations    int
	config           json.RawMessage // nil: 204
	revision         string
	omitETag         bool
	lastIfNoneM      string
	restBeats        int
	deregistered     int
}

func newFakeBackend(t *testing.T) *fakeBackend {
	b := &fakeBackend{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/fleet/register/{scope}/", b.register)
	mux.HandleFunc("GET /api/fleet/config/{id}", b.getConfig)
	mux.HandleFunc("POST /api/fleet/heartbeat/{id}", b.heartbeat)
	mux.HandleFunc("DELETE /api/fleet/deregister/{id}", b.deregister)
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != testToken {
			http.Error(w, `{"error":"bad token"}`, http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(b.srv.Close)
	return b
}

func (b *fakeBackend) register(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.registerAttempts++
	if b.registerStatus != 0 {
		http.Error(w, `{"error":"nope"}`, b.registerStatus)
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b.nextID++
	b.current = fmt.Sprintf("%032x", b.nextID)
	b.lastRegPath, b.lastReg = r.URL.Path, req
	b.registrations++
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(RegisterResponse{InstanceID: b.current})
}

func (b *fakeBackend) getConfig(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r.PathValue("id") != b.current || b.current == "" {
		http.Error(w, `{"error":"unknown"}`, http.StatusNotFound)
		return
	}
	if b.config == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	b.lastIfNoneM = r.Header.Get("If-None-Match")
	if !b.omitETag {
		w.Header().Set("ETag", `"`+b.revision+`"`)
		w.Header().Set(HeaderConfigRevision, b.revision)
		if b.lastIfNoneM == `"`+b.revision+`"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b.config)
}

func (b *fakeBackend) heartbeat(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r.PathValue("id") != b.current || b.current == "" {
		http.Error(w, `{"error":"unknown"}`, http.StatusNotFound)
		return
	}
	b.restBeats++
	w.WriteHeader(http.StatusNoContent)
}

func (b *fakeBackend) deregister(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.deregistered++
	if r.PathValue("id") != b.current || b.current == "" {
		http.Error(w, `{"error":"unknown"}`, http.StatusNotFound)
		return
	}
	b.current = ""
	w.WriteHeader(http.StatusNoContent)
}

func (b *fakeBackend) setConfig(blob, revision string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if blob == "" {
		b.config = nil
	} else {
		b.config = json.RawMessage(blob)
	}
	b.revision = revision
}

// rotate invalidates the current instance id, as a deregistration by an
// operator or a re-registration from elsewhere would.
func (b *fakeBackend) rotate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = ""
}

func (b *fakeBackend) get(f func(b *fakeBackend) int) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return f(b)
}

func newTestClient(t *testing.T, b *fakeBackend, mod func(*Options)) *Client {
	t.Helper()
	opts := Options{
		Config: Config{
			URL:                b.srv.URL,
			Token:              testToken,
			Cluster:            "fritz",
			Hostname:           "f0101",
			HeartbeatInterval:  "20ms",
			ConfigPollInterval: "20ms",
		},
		ServiceType: ServiceMetricCollector,
		Meta:        map[string]string{"version": "1.0"},
	}
	if mod != nil {
		mod(&opts)
	}
	c, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRegister_Scope(t *testing.T) {
	tests := []struct {
		name, cluster, wantPath string
	}{
		{"cluster", "fritz", "/api/fleet/register/cluster/"},
		{"infra", "", "/api/fleet/register/infra/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newFakeBackend(t)
			c := newTestClient(t, b, func(o *Options) { o.Config.Cluster = tt.cluster })
			if _, err := c.Register(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !c.Registered() {
				t.Error("not registered")
			}
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.lastRegPath != tt.wantPath {
				t.Errorf("path = %q, want %q", b.lastRegPath, tt.wantPath)
			}
			want := RegisterRequest{Cluster: tt.cluster, Hostname: "f0101", ServiceType: ServiceMetricCollector, MetaData: map[string]string{"version": "1.0"}}
			if fmt.Sprint(b.lastReg) != fmt.Sprint(want) {
				t.Errorf("body = %+v, want %+v", b.lastReg, want)
			}
		})
	}
}

func TestRegister_Unauthorized(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, func(o *Options) { o.Config.Token = "wrong" })
	_, err := c.Register(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	var se *StatusError
	if !errors.As(err, &se) || se.Code != http.StatusUnauthorized {
		t.Errorf("want StatusError 401, got %v", err)
	}
}

func TestPullConfig_StatusSequence(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	ctx := context.Background()

	if _, _, err := c.PullConfig(ctx); !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("before register: err = %v, want ErrNotRegistered", err)
	}
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		name        string
		setup       func()
		wantConfig  string
		wantChanged bool
		wantSource  Source
		wantErr     error
	}{
		{"204 without config", nil, "", false, SourceNone, nil},
		{"200 first config", func() { b.setConfig(`{"a":1}`, "11") }, `{"a":1}`, true, SourceFleet, nil},
		{"304 unchanged", nil, `{"a":1}`, false, SourceFleet, nil},
		{"200 new config", func() { b.setConfig(`{"a":2}`, "22") }, `{"a":2}`, true, SourceFleet, nil},
		{"204 after 200", func() { b.setConfig("", "") }, "", true, SourceNone, nil},
		{"404 unknown", b.rotate, "", false, SourceNone, ErrUnknownInstance},
	}
	for _, s := range steps {
		if s.setup != nil {
			s.setup()
		}
		u, changed, err := c.PullConfig(ctx)
		if !errors.Is(err, s.wantErr) {
			t.Fatalf("%s: err = %v, want %v", s.name, err, s.wantErr)
		}
		if string(u.Config) != s.wantConfig || changed != s.wantChanged || u.Source != s.wantSource {
			t.Errorf("%s: got (%s, changed=%v, %s), want (%s, %v, %s)",
				s.name, u.Config, changed, u.Source, s.wantConfig, s.wantChanged, s.wantSource)
		}
	}
	if c.Registered() {
		t.Error("client should forget the instance id after 404")
	}
}

func TestPullConfig_SendsETag(t *testing.T) {
	b := newFakeBackend(t)
	b.setConfig(`{"a":1}`, "7")
	c := newTestClient(t, b, nil)
	ctx := context.Background()
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}
	u, _, err := c.PullConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if u.Revision != "7" {
		t.Errorf("revision = %q, want 7", u.Revision)
	}
	if _, _, err := c.PullConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if got := b.get(func(b *fakeBackend) int { return len(b.lastIfNoneM) }); got == 0 {
		t.Error("second pull did not send If-None-Match")
	}
}

func TestPullConfig_NoETagSameBody(t *testing.T) {
	b := newFakeBackend(t)
	b.setConfig(`{"a":1}`, "")
	b.omitETag = true
	c := newTestClient(t, b, nil)
	ctx := context.Background()
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}
	if _, changed, _ := c.PullConfig(ctx); !changed {
		t.Error("first pull should report a change")
	}
	if _, changed, _ := c.PullConfig(ctx); changed {
		t.Error("identical body without ETag must not report a change")
	}
}

func TestHeartbeat_REST(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	ctx := context.Background()
	if err := c.Heartbeat(ctx); !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("err = %v, want ErrNotRegistered", err)
	}
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Heartbeat(ctx); err != nil {
		t.Fatal(err)
	}
	if n := b.get(func(b *fakeBackend) int { return b.restBeats }); n != 1 {
		t.Errorf("rest heartbeats = %d, want 1", n)
	}
	b.rotate()
	if err := c.Heartbeat(ctx); !errors.Is(err, ErrUnknownInstance) {
		t.Errorf("err = %v, want ErrUnknownInstance", err)
	}
	if c.Registered() {
		t.Error("client should forget the instance id after 404")
	}
}

func TestDeregister_Idempotent(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	ctx := context.Background()

	if err := c.Deregister(ctx); err != nil {
		t.Fatalf("deregister without registration: %v", err)
	}
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}
	b.rotate()
	if err := c.Deregister(ctx); err != nil {
		t.Fatalf("deregister of unknown instance: %v", err)
	}
	if c.Registered() {
		t.Error("client should forget the instance id")
	}
}

func TestStatusError_NoInstanceIDInMessage(t *testing.T) {
	b := newFakeBackend(t)
	c := newTestClient(t, b, nil)
	ctx := context.Background()
	if _, err := c.Register(ctx); err != nil {
		t.Fatal(err)
	}
	id := c.id()
	b.rotate()
	_, _, err := c.PullConfig(ctx)
	if err == nil || strings.Contains(err.Error(), id) {
		t.Errorf("error %q must not contain the instance id", err)
	}
}

// waitFor polls cond until it holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
