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
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// Source says where a configuration update came from.
type Source int

const (
	// SourceNone means no fleet configuration applies to this member; run
	// with the local or built-in configuration.
	SourceNone Source = iota
	// SourceFleet is a configuration received from cc-backend.
	SourceFleet
	// SourceCache is the last known configuration, read from the cache file
	// because cc-backend was unreachable.
	SourceCache
)

func (s Source) String() string {
	switch s {
	case SourceFleet:
		return "fleet"
	case SourceCache:
		return "cache"
	}
	return "none"
}

// Update is a configuration delivered to the application.
type Update struct {
	// Config is the merged configuration object, or nil when no fleet
	// configuration applies (Source is SourceNone).
	Config   json.RawMessage
	Revision string
	Source   Source
}

// Registration retry backoff.
const (
	minBackoff = time.Second
	maxBackoff = 5 * time.Minute
)

// retryState spaces out registration attempts after failures, so a rejected
// token does not hammer cc-backend on every tick.
type retryState struct {
	failures int
	next     time.Time
}

func (r *retryState) wait(now time.Time) bool { return now.Before(r.next) }

func (r *retryState) fail(now time.Time) {
	d := minBackoff << min(r.failures, 16)
	r.failures++
	r.next = now.Add(min(d, maxBackoff))
}

func (r *retryState) reset() { *r = retryState{} }

// Configs delivers configuration changes found by Run. The channel holds at
// most one value; an unread update is replaced by the newer one.
func (c *Client) Configs() <-chan Update { return c.configs }

// Rosters delivers discovery rosters. Each one is the full provider list, not a
// delta. The channel holds at most one value; an unread roster is replaced by
// the newer one.
func (c *Client) Rosters() <-chan []Provider { return c.rosters }

// Bootstrap gets the configuration to start with. It loads the cache file (if
// configured), then registers and pulls within timeout.
//
// On success the returned update has Source SourceFleet, or SourceNone when
// no configuration is authored for this member. On failure the cached
// configuration (SourceCache) or an empty update (SourceNone) is returned
// together with the error: ErrUnreachable for transport errors, a
// *StatusError otherwise. The application can start in either case; Run keeps
// trying to register. Call Bootstrap at most once, before Run.
func (c *Client) Bootstrap(ctx context.Context, timeout time.Duration) (Update, error) {
	cached, haveCache := c.loadCache()
	fallback := Update{Source: SourceNone}
	if haveCache {
		fallback = cached
	}

	bctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if !c.Registered() {
		if _, err := c.Register(bctx); err != nil {
			c.regRetry.fail(time.Now())
			c.report("register", err)
			return fallback, classify(err)
		}
	}
	u, _, err := c.PullConfig(bctx)
	if err != nil {
		c.report("config", err)
		return fallback, classify(err)
	}
	return u, nil
}

// classify wraps transport errors in ErrUnreachable and leaves HTTP status
// errors as they are.
func classify(err error) error {
	var se *StatusError
	if errors.As(err, &se) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrUnreachable, err)
}

// Run keeps the membership alive until ctx is cancelled or Close is called. It
// pulls the configuration at once and then every config-poll-interval, sends a
// heartbeat every heartbeat-interval, (re-)registers whenever needed, and
// subscribes to the discovery roster once the NATS client is connected.
// Changes are delivered on Configs and Rosters. Run returns nil on
// cancellation and an error only if it is already running or closed.
func (c *Client) Run(ctx context.Context) error {
	c.runMu.Lock()
	if c.running {
		c.runMu.Unlock()
		return errors.New("fleet: Run already running")
	}
	if c.closed.Load() {
		c.runMu.Unlock()
		return errors.New("fleet: client closed")
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	c.running, c.cancel, c.runDone = true, cancel, done
	c.runMu.Unlock()

	defer func() {
		cancel()
		close(done)
		c.runMu.Lock()
		c.running = false
		c.runMu.Unlock()
	}()

	heartbeat := time.NewTicker(c.intervals.heartbeat)
	defer heartbeat.Stop()
	poll := time.NewTicker(c.intervals.poll)
	defer poll.Stop()

	c.pollTick(ctx)
	c.heartbeatTick(ctx)
	c.subscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			c.heartbeatTick(ctx)
			c.subscribe()
		case <-poll.C:
			c.pollTick(ctx)
		}
	}
}

// Close stops Run, waits for it to return and deregisters, so the member
// leaves the discovery rosters at once. ctx bounds the wait and the
// deregistration. Skipping Close is not fatal: cc-backend marks the member
// stale after stale-after.
func (c *Client) Close(ctx context.Context) error {
	c.closed.Store(true)

	c.runMu.Lock()
	running, cancel, done := c.running, c.cancel, c.runDone
	c.runMu.Unlock()
	if running {
		cancel()
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return c.Deregister(ctx)
}

// ensureRegistered registers when no instance id is held, honouring the retry
// backoff. It reports whether the client is registered afterwards.
func (c *Client) ensureRegistered(ctx context.Context) bool {
	if c.Registered() {
		return true
	}
	now := time.Now()
	if c.regRetry.wait(now) {
		return false
	}
	if _, err := c.Register(ctx); err != nil {
		if ctx.Err() == nil {
			c.regRetry.fail(now)
			c.report("register", err)
		}
		return false
	}
	c.regRetry.reset()
	c.report("register", nil)
	return true
}

func (c *Client) heartbeatTick(ctx context.Context) {
	if !c.ensureRegistered(ctx) {
		return
	}
	err := c.Heartbeat(ctx)
	if errors.Is(err, ErrUnknownInstance) {
		cclog.ComponentInfo(component, "cc-backend does not know this instance any more, registering again")
		c.reregister(ctx)
		return
	}
	if ctx.Err() == nil {
		c.report("heartbeat", err)
	}
}

func (c *Client) pollTick(ctx context.Context) {
	if !c.ensureRegistered(ctx) {
		return
	}
	err := c.pullAndDeliver(ctx)
	if errors.Is(err, ErrUnknownInstance) {
		cclog.ComponentInfo(component, "cc-backend does not know this instance any more, registering again")
		c.reregister(ctx)
		return
	}
	if ctx.Err() == nil {
		c.report("config", err)
	}
}

// reregister registers again right away and pulls once, without waiting for
// the next tick. It does not recurse on a second 404.
func (c *Client) reregister(ctx context.Context) {
	if !c.ensureRegistered(ctx) {
		return
	}
	if err := c.Heartbeat(ctx); ctx.Err() == nil {
		c.report("heartbeat", err)
	}
	if err := c.pullAndDeliver(ctx); ctx.Err() == nil {
		c.report("config", err)
	}
}

func (c *Client) pullAndDeliver(ctx context.Context) error {
	u, changed, err := c.PullConfig(ctx)
	if err != nil {
		return err
	}
	if changed {
		cclog.ComponentInfof(component, "configuration changed (source %s, revision %q)", u.Source, u.Revision)
		offerLatest(c.configs, u)
	}
	return nil
}

// subscribe subscribes to the discovery roster once a connected NATS client is
// available, and again if the provider starts returning a different client.
func (c *Client) subscribe() {
	nc := c.connectedNATS()
	if nc == nil || nc == c.subscribedTo {
		return
	}
	subject := DiscoverySubject(c.discPrefix, Bucket(c.cluster), c.svcType)
	err := nc.Subscribe(subject, func(_ string, data []byte) {
		if c.closed.Load() {
			return
		}
		providers, err := DecodeRoster(data)
		if err != nil {
			cclog.ComponentWarnf(component, "ignoring malformed discovery roster: %v", err)
			return
		}
		offerLatest(c.rosters, providers)
	})
	c.report("discovery", err)
	if err == nil {
		c.subscribedTo = nc
	}
}

// report logs the first failure of an operation as a warning, repeats at
// debug level while it keeps failing, and logs the recovery.
func (c *Client) report(op string, err error) {
	if err == nil {
		if c.failing[op] {
			cclog.ComponentInfof(component, "%s succeeded again", op)
			delete(c.failing, op)
		}
		return
	}
	if errors.Is(err, ErrNotRegistered) {
		return
	}
	if c.failing[op] {
		cclog.ComponentDebugf(component, "%s still failing: %v", op, err)
		return
	}
	c.failing[op] = true
	cclog.ComponentWarnf(component, "%s failed: %v", op, err)
}

// offerLatest sends v without blocking. If the channel is full, the stale value
// is dropped in favour of v.
func offerLatest[T any](ch chan T, v T) {
	for {
		select {
		case ch <- v:
			return
		default:
		}
		select {
		case <-ch:
		default:
		}
	}
}
