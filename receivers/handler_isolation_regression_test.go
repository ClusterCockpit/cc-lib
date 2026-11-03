// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package receivers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

type isolationConstructor func(string, json.RawMessage) (Receiver, error)

func isolatedHandler(t *testing.T, constructor isolationConstructor, name, path string) (http.Handler, chan lp.CCMessage) {
	t.Helper()
	config, err := json.Marshal(map[string]any{
		"address": "127.0.0.1", "port": "0", "path": path,
		"username": "synthetic-" + name, "password": "fixture-" + name,
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := constructor(name, config)
	if err != nil {
		t.Fatal(err)
	}
	sink := make(chan lp.CCMessage, 1)
	r.SetSink(sink)
	var handler http.Handler
	switch receiver := r.(type) {
	case *HttpReceiver:
		handler = receiver.server.Handler
	case *EECPTReceiver:
		handler = receiver.server.Handler
	default:
		t.Fatal("unexpected receiver fixture")
	}
	if handler == nil {
		handler = http.DefaultServeMux // Same fallback as http.Server.Serve.
	}
	return handler, sink
}

func isolatedRequest(handler http.Handler, path, identity string) int {
	request := httptest.NewRequest(http.MethodPost, "http://receiver.invalid"+path,
		strings.NewReader("audit_metric,type=node,hostname=fixture value=42i 1000000000"))
	request.SetBasicAuth("synthetic-"+identity, "fixture-"+identity)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func TestReceiverInstancesOwnHandlers(t *testing.T) {
	for name, constructor := range map[string]isolationConstructor{"http": NewHttpReceiver, "eecpt": NewEECPTReceiver} {
		t.Run(name, func(t *testing.T) {
			previous := http.DefaultServeMux
			http.DefaultServeMux = http.NewServeMux()
			defer func() {
				http.DefaultServeMux = previous
				if value := recover(); value != nil {
					t.Errorf("constructors with independent server instances conflicted: %v", value)
				}
			}()
			a, aSink := isolatedHandler(t, constructor, "a", "/audit-write")
			b, bSink := isolatedHandler(t, constructor, "b", "/audit-write")
			if isolatedRequest(a, "/audit-write", "b") != http.StatusUnauthorized ||
				isolatedRequest(b, "/audit-write", "a") != http.StatusUnauthorized {
				t.Fatal("independent handler accepted the other identity")
			}
			if isolatedRequest(a, "/audit-write", "a") != http.StatusOK || len(aSink) != 1 || len(bSink) != 0 {
				t.Fatal("first instance did not route only into its own sink")
			}
			if isolatedRequest(b, "/audit-write", "b") != http.StatusOK || len(bSink) != 1 {
				t.Fatal("second instance did not route into its own sink")
			}
		})
	}
}

func TestReceiverInstancesDoNotExposeOtherPaths(t *testing.T) {
	for name, constructor := range map[string]isolationConstructor{"http": NewHttpReceiver, "eecpt": NewEECPTReceiver} {
		t.Run(name, func(t *testing.T) {
			previous := http.DefaultServeMux
			http.DefaultServeMux = http.NewServeMux()
			defer func() { http.DefaultServeMux = previous }()
			a, aSink := isolatedHandler(t, constructor, "a", "/audit-a")
			b, bSink := isolatedHandler(t, constructor, "b", "/audit-b")
			if isolatedRequest(a, "/audit-b", "b") != http.StatusNotFound ||
				isolatedRequest(b, "/audit-a", "a") != http.StatusNotFound {
				t.Fatal("a receiver exposed another receiver's route")
			}
			if len(aSink) != 0 || len(bSink) != 0 {
				t.Fatal("cross-route request reached another receiver's sink")
			}
		})
	}
}
