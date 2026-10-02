// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"

	"github.com/ClusterCockpit/cc-lib/v2/nats"
)

func startNATS(t *testing.T) *nats.Client {
	t.Helper()
	ns, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: server.RANDOM_PORT})
	if err != nil {
		t.Fatal(err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(4 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(ns.Shutdown)

	nc, err := nats.NewClient(&nats.NatsConfig{Address: ns.ClientURL()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestRun_NATSHeartbeatAndDiscovery(t *testing.T) {
	const hbSubject = "cc.fleet.event"
	b := newFakeBackend(t)
	nc := startNATS(t)

	var mu sync.Mutex
	var beats []string
	if err := nc.Subscribe(hbSubject, func(_ string, data []byte) {
		ids, err := DecodeHeartbeats(data)
		if err != nil {
			t.Errorf("decoding heartbeat: %v", err)
			return
		}
		mu.Lock()
		beats = append(beats, ids...)
		mu.Unlock()
	}); err != nil {
		t.Fatal(err)
	}

	// The NATS client becomes available only after start, as it would when
	// its settings come with the fleet configuration.
	var provided atomic.Pointer[nats.Client]
	c := newTestClient(t, b, func(o *Options) {
		o.Config.HeartbeatSubject = hbSubject
		o.NATS = provided.Load
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runErr := make(chan error, 1)
	go func() { runErr <- c.Run(ctx) }()

	waitFor(t, "REST heartbeats while NATS is unavailable", func() bool {
		return b.get(func(b *fakeBackend) int { return b.restBeats }) > 0
	})

	provided.Store(nc)
	waitFor(t, "NATS heartbeats", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(beats) > 0
	})
	mu.Lock()
	if beats[0] != c.id() {
		t.Error("heartbeat carries a different instance id")
	}
	mu.Unlock()

	restBefore := b.get(func(b *fakeBackend) int { return b.restBeats })
	time.Sleep(100 * time.Millisecond)
	if restAfter := b.get(func(b *fakeBackend) int { return b.restBeats }); restAfter != restBefore {
		t.Errorf("REST heartbeats continued (%d → %d) although NATS is connected", restBefore, restAfter)
	}

	want := []Provider{{Type: ServiceBackend, Hostname: "mgmt01", State: "active", Meta: map[string]string{"port": "8080"}}}
	roster, err := EncodeRoster("fritz", ServiceMetricCollector, want, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	subject := DiscoverySubject("", "fritz", ServiceMetricCollector)

	// The subscription is set up on the next heartbeat tick, so publish until
	// the roster arrives (rosters are re-published periodically in reality).
	deadline := time.After(5 * time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for done := false; !done; {
		select {
		case got := <-c.Rosters():
			if !reflect.DeepEqual(got, want) {
				t.Errorf("roster = %+v, want %+v", got, want)
			}
			done = true
		case <-tick.C:
			if err := nc.Publish(subject, roster); err != nil {
				t.Fatal(err)
			}
		case <-deadline:
			t.Fatal("no roster delivered")
		}
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	if err := c.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
	if err := <-runErr; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
