// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"encoding/json"
	"testing"
)

func TestParseConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"minimal", `{"url":"https://cc.example.org"}`, false},
		{"full", `{"url":"http://ccb:8080","token":"t","cluster":"fritz","hostname":"f0101",
			"heartbeat-interval":"15s","config-poll-interval":"1m","heartbeat-subject":"cc.fleet.event",
			"discovery-subject-prefix":"cc.fleet.discovery","cache-path":"/tmp/x","timeout":"5s"}`, false},
		{"empty", ``, true},
		{"missing url", `{"cluster":"fritz"}`, true},
		{"unknown key", `{"url":"http://ccb","bogus":1}`, true},
		{"bad duration", `{"url":"http://ccb","heartbeat-interval":"often"}`, true},
		{"negative duration", `{"url":"http://ccb","timeout":"-1s"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig(json.RawMessage(tt.raw))
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{"ok", Options{Config: Config{URL: "http://ccb", Hostname: "h1"}, ServiceType: ServiceMetricStore}, false},
		{"fqdn hostname", Options{Config: Config{URL: "http://ccb", Hostname: "h1.example.org"}, ServiceType: ServiceMetricStore}, false},
		{"unknown type", Options{Config: Config{URL: "http://ccb", Hostname: "h1"}, ServiceType: "xyz"}, true},
		{"bad url", Options{Config: Config{URL: "ccb:8080", Hostname: "h1"}, ServiceType: ServiceMetricStore}, true},
		{"slash in hostname", Options{Config: Config{URL: "http://ccb", Hostname: "a/b"}, ServiceType: ServiceMetricStore}, true},
		{"dotdot cluster", Options{Config: Config{URL: "http://ccb", Hostname: "h1", Cluster: ".."}, ServiceType: ServiceMetricStore}, true},
		{"bad duration", Options{Config: Config{URL: "http://ccb", Hostname: "h1", Timeout: "x"}, ServiceType: ServiceMetricStore}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew_TokenFromEnv(t *testing.T) {
	t.Setenv(EnvToken, "from-env")
	c, err := New(Options{Config: Config{URL: "http://ccb", Hostname: "h1", Token: "from-config"}, ServiceType: ServiceMetricStore})
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "from-env" {
		t.Errorf("token = %q, want from-env", c.token)
	}
}
