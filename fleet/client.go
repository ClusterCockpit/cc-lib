// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/nats"
	"github.com/ClusterCockpit/cc-lib/v2/util"
)

const component = "Fleet"

// maxConfigSize bounds the configuration body read from cc-backend.
const maxConfigSize = 16 << 20

var (
	// ErrUnknownInstance is returned when cc-backend does not know the
	// instance id (never registered, rotated or deregistered). The client
	// forgets the id; registering again issues a new one.
	ErrUnknownInstance = errors.New("fleet: unknown or deregistered instance")
	// ErrUnauthorized is returned for 401 and 403: the token is missing,
	// invalid or lacks the api role, or the source IP is not allowlisted.
	ErrUnauthorized = errors.New("fleet: request not authorized")
	// ErrNotRegistered is returned by calls that need an instance id before
	// Register succeeded.
	ErrNotRegistered = errors.New("fleet: not registered")
	// ErrUnreachable is returned by Bootstrap when cc-backend could not be
	// reached at all.
	ErrUnreachable = errors.New("fleet: cc-backend unreachable")
)

// StatusError reports an unexpected HTTP status. It unwraps to
// ErrUnknownInstance for 404 and to ErrUnauthorized for 401 and 403.
type StatusError struct {
	Op     string // register, config, heartbeat, deregister
	Code   int
	Status string
	Detail string // start of the response body, if any
}

func (e *StatusError) Error() string {
	msg := fmt.Sprintf("fleet %s: unexpected status %s", e.Op, e.Status)
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

func (e *StatusError) Unwrap() error {
	switch e.Code {
	case http.StatusNotFound:
		return ErrUnknownInstance
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	}
	return nil
}

// Options configures a Client.
type Options struct {
	Config      Config
	ServiceType ServiceType
	// Meta is registration metadata. It is broadcast unauthenticated in
	// discovery rosters: connection hints only, never secrets.
	Meta map[string]string
	// NATS returns the NATS client used for heartbeats and discovery. It is
	// called on every use and may return nil until a connection exists, for
	// example because the NATS settings arrive with the fleet configuration.
	// Without it the client works over REST only.
	NATS func() *nats.Client
	// HTTPClient overrides the HTTP client, e.g. for custom TLS. The default
	// uses Config.Timeout.
	HTTPClient *http.Client
}

// Client is a fleet member. Its methods are safe for concurrent use.
type Client struct {
	baseURL    string
	token      string
	svcType    ServiceType
	cluster    string
	hostname   string
	meta       map[string]string
	natsFn     func() *nats.Client
	http       *http.Client
	intervals  durations
	hbSubject  string
	discPrefix string
	cachePath  string

	mu         sync.Mutex
	instanceID string
	etag       string
	revision   string
	blob       json.RawMessage

	configs chan Update
	rosters chan []Provider
	closed  atomic.Bool

	runMu   sync.Mutex
	running bool
	cancel  context.CancelFunc
	runDone chan struct{}

	// Owned by the Run goroutine (and Bootstrap, which precedes it).
	regRetry     retryState
	failing      map[string]bool
	subscribedTo *nats.Client
}

// New validates the options and creates a client. It does not contact
// cc-backend. The token is resolved from CC_FLEET_TOKEN, CC_FLEET_TOKEN_FILE
// or Config.Token, in that order.
func New(opts Options) (*Client, error) {
	cfg := opts.Config
	if !opts.ServiceType.Valid() {
		return nil, fmt.Errorf("fleet: unknown service type %q", opts.ServiceType)
	}
	u, err := url.Parse(cfg.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("fleet: invalid url %q", cfg.URL)
	}
	d, err := cfg.durations()
	if err != nil {
		return nil, err
	}
	token, err := util.SecretFromEnv(EnvToken, cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("fleet: resolving token: %w", err)
	}

	hostname := cfg.Hostname
	if hostname == "" {
		if hostname, err = os.Hostname(); err != nil {
			return nil, fmt.Errorf("fleet: determining hostname: %w", err)
		}
	}
	if !safePathComponent(hostname) {
		return nil, fmt.Errorf("fleet: invalid hostname %q", hostname)
	}
	if cfg.Cluster != "" && !safePathComponent(cfg.Cluster) {
		return nil, fmt.Errorf("fleet: invalid cluster %q", cfg.Cluster)
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: d.timeout}
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		token:      token,
		svcType:    opts.ServiceType,
		cluster:    cfg.Cluster,
		hostname:   hostname,
		meta:       opts.Meta,
		natsFn:     opts.NATS,
		http:       httpClient,
		intervals:  d,
		hbSubject:  cfg.HeartbeatSubject,
		discPrefix: cfg.DiscoverySubjectPrefix,
		cachePath:  cfg.CachePath,
		configs:    make(chan Update, 1),
		rosters:    make(chan []Provider, 1),
		failing:    make(map[string]bool),
	}, nil
}

// Scope returns the registration scope, derived from the configured cluster.
func (c *Client) Scope() Scope {
	if c.cluster == "" {
		return ScopeInfra
	}
	return ScopeCluster
}

// Registered reports whether the client holds an instance id.
func (c *Client) Registered() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.instanceID != ""
}

func (c *Client) id() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.instanceID
}

// forget drops the instance id, but only if it is still the one that failed,
// so a concurrent re-registration is not undone.
func (c *Client) forget(id string) {
	c.mu.Lock()
	if c.instanceID == id {
		c.instanceID = ""
	}
	c.mu.Unlock()
}

// Register registers the member and stores the new instance id. Any previous
// id becomes invalid. The cached ETag is kept, so an unchanged configuration
// is not delivered again after re-registration.
func (c *Client) Register(ctx context.Context) (RegisterResponse, error) {
	var reg RegisterResponse
	body, err := json.Marshal(RegisterRequest{
		Cluster:     c.cluster,
		Hostname:    c.hostname,
		ServiceType: c.svcType,
		MetaData:    c.meta,
	})
	if err != nil {
		return reg, err
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/fleet/register/"+string(c.Scope())+"/", body, "")
	if err != nil {
		return reg, fmt.Errorf("fleet register: %w", err)
	}
	defer drain(resp)

	if resp.StatusCode != http.StatusCreated {
		return reg, statusError("register", resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return reg, fmt.Errorf("fleet register: decoding response: %w", err)
	}
	if reg.InstanceID == "" {
		return reg, errors.New("fleet register: response without instance id")
	}

	c.mu.Lock()
	c.instanceID = reg.InstanceID
	c.mu.Unlock()
	return reg, nil
}

// PullConfig polls the configuration. changed is true when the delivered
// configuration differs from the previous one, including the transition to
// "no configuration" (204, Update.Config nil). An unchanged answer (304, or
// 200 with an identical body) returns the current configuration with changed
// false.
func (c *Client) PullConfig(ctx context.Context) (Update, bool, error) {
	c.mu.Lock()
	id, etag := c.instanceID, c.etag
	c.mu.Unlock()
	if id == "" {
		return Update{}, false, ErrNotRegistered
	}

	resp, err := c.do(ctx, http.MethodGet, "/api/fleet/config/"+id, nil, etag)
	if err != nil {
		return Update{}, false, fmt.Errorf("fleet config: %w", err)
	}
	defer drain(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		blob, err := io.ReadAll(io.LimitReader(resp.Body, maxConfigSize+1))
		if err != nil {
			return Update{}, false, fmt.Errorf("fleet config: reading body: %w", err)
		}
		if len(blob) > maxConfigSize {
			return Update{}, false, fmt.Errorf("fleet config: body exceeds %d bytes", maxConfigSize)
		}
		if !json.Valid(blob) {
			return Update{}, false, errors.New("fleet config: body is not valid JSON")
		}
		newETag := resp.Header.Get("ETag")
		rev := resp.Header.Get(HeaderConfigRevision)
		if rev == "" {
			rev = strings.Trim(strings.TrimPrefix(newETag, "W/"), `"`)
		}

		c.mu.Lock()
		changed := c.blob == nil || !bytes.Equal(c.blob, blob)
		c.etag, c.revision, c.blob = newETag, rev, blob
		c.mu.Unlock()

		if changed {
			c.writeCache(newETag, rev, blob)
		}
		return Update{Config: blob, Revision: rev, Source: SourceFleet}, changed, nil

	case http.StatusNotModified:
		c.mu.Lock()
		u := Update{Config: c.blob, Revision: c.revision, Source: SourceFleet}
		c.mu.Unlock()
		return u, false, nil

	case http.StatusNoContent:
		c.mu.Lock()
		changed := c.blob != nil
		c.etag, c.revision, c.blob = "", "", nil
		c.mu.Unlock()

		if changed {
			c.removeCache()
		}
		return Update{Source: SourceNone}, changed, nil

	default:
		if resp.StatusCode == http.StatusNotFound {
			c.forget(id)
		}
		return Update{}, false, statusError("config", resp)
	}
}

// Heartbeat marks the member alive. It publishes on the heartbeat subject when
// one is configured and the NATS client is connected, and falls back to REST
// otherwise, so heartbeats are never buffered silently by a disconnected
// client. Only the REST path can detect an unknown instance; with NATS,
// Run relies on the config poll for that.
func (c *Client) Heartbeat(ctx context.Context) error {
	id := c.id()
	if id == "" {
		return ErrNotRegistered
	}

	if nc := c.connectedNATS(); nc != nil && c.hbSubject != "" {
		msg, err := EncodeHeartbeat(time.Now(), id)
		if err != nil {
			return err
		}
		if err := nc.Publish(c.hbSubject, msg); err == nil {
			return nil
		}
		// Fall through to REST: the connection dropped in between.
	}

	resp, err := c.do(ctx, http.MethodPost, "/api/fleet/heartbeat/"+id, nil, "")
	if err != nil {
		return fmt.Errorf("fleet heartbeat: %w", err)
	}
	defer drain(resp)

	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		c.forget(id)
	}
	return statusError("heartbeat", resp)
}

// Deregister removes the member from the fleet and from discovery rosters at
// once. It is idempotent: without an instance id, or when cc-backend does not
// know the id any more, it returns nil.
func (c *Client) Deregister(ctx context.Context) error {
	id := c.id()
	if id == "" {
		return nil
	}

	resp, err := c.do(ctx, http.MethodDelete, "/api/fleet/deregister/"+id, nil, "")
	if err != nil {
		return fmt.Errorf("fleet deregister: %w", err)
	}
	defer drain(resp)

	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusNotFound:
		c.forget(id)
		return nil
	}
	return statusError("deregister", resp)
}

func (c *Client) connectedNATS() *nats.Client {
	if c.natsFn == nil {
		return nil
	}
	nc := c.natsFn()
	if nc == nil || !nc.IsConnected() {
		return nil
	}
	return nc
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, ifNoneMatch string) (*http.Response, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	return c.http.Do(req)
}

func statusError(op string, resp *http.Response) error {
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return &StatusError{
		Op:     op,
		Code:   resp.StatusCode,
		Status: resp.Status,
		Detail: strings.TrimSpace(string(detail)),
	}
}

// drain discards the rest of the body so the connection can be reused.
func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}
