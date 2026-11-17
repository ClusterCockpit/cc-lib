// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sinks

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func newProcessingTestNatsSink(t *testing.T) (*NatsSink, *nats.Subscription) {
	t.Helper()
	ns, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatal(err)
	}
	go ns.Start()
	t.Cleanup(func() { ns.Shutdown(); ns.WaitForShutdown() })
	if !ns.ReadyForConnections(4 * time.Second) {
		t.Fatal("own NATS broker not ready")
	}
	observer, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	sub, err := observer.SubscribeSync("audit.processed")
	if err != nil {
		t.Fatal(err)
	}
	if err = observer.FlushTimeout(time.Second); err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(ns.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(NatsSinkConfig{defaultSinkConfig: defaultSinkConfig{MetaAsTags: []string{"unit"}}, Host: host, Port: port, Subject: "audit.processed", FlushDelay: "1h", Precision: "s"})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := NewNatsSink("audit-processing", config)
	if err != nil {
		t.Fatal(err)
	}
	sink := instance.(*NatsSink)
	// TestClose flushes and closes explicitly; avoid closing the sink twice.
	t.Cleanup(func() {
		if !sink.client.IsClosed() {
			sink.Close()
		}
	})
	return sink, sub
}

func TestNatsSinkEncodesProcessedMessage(t *testing.T) {
	sink, sub := newProcessingTestNatsSink(t)
	if err := sink.mp.AddRenameMetricByName("before", "after"); err != nil {
		t.Fatal(err)
	}
	input, err := lp.NewMetric("before", map[string]string{"type": "node"}, map[string]string{"unit": "W"}, float64(7), time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err = sink.Write(input); err != nil {
		t.Fatal(err)
	}
	if err = sink.Flush(); err != nil {
		t.Fatal(err)
	}
	msg, err := sub.NextMsg(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(msg.Data), "after,type=node,unit=W value=7 100\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if input.Name() != "before" || input.Tags()["unit"] != "" {
		t.Fatal("processing mutated caller-owned input")
	}
}
