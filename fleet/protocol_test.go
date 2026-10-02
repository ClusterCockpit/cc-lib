// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"reflect"
	"strings"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

func TestEncodeHeartbeat_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
	}{
		{"single", []string{"3f1c9a2b7d4e6f8a0b1c2d3e4f5a6b7c"}},
		{"batched", []string{"aaaa", "bbbb", "cccc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := EncodeHeartbeat(time.Now(), tt.ids...)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(data), "\n"); got != len(tt.ids) {
				t.Errorf("got %d lines, want %d: %q", got, len(tt.ids), data)
			}
			ids, err := DecodeHeartbeats(data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ids, tt.ids) {
				t.Errorf("got %v, want %v", ids, tt.ids)
			}
		})
	}
}

func TestEncodeHeartbeat_Invalid(t *testing.T) {
	if _, err := EncodeHeartbeat(time.Now()); err == nil {
		t.Error("expected error without ids")
	}
	if _, err := EncodeHeartbeat(time.Now(), ""); err == nil {
		t.Error("expected error for empty id")
	}
}

func TestParseHeartbeat_Rejects(t *testing.T) {
	tests := []struct {
		name        string
		measurement string
		tags        map[string]string
		payload     string
	}{
		{"wrong measurement", "other", map[string]string{"function": "heartbeat"}, `{"instanceId":"a"}`},
		{"wrong function", "fleet", map[string]string{"function": "register"}, `{"instanceId":"a"}`},
		{"missing function", "fleet", nil, `{"instanceId":"a"}`},
		{"unknown key", "fleet", map[string]string{"function": "heartbeat"}, `{"instanceId":"a","x":1}`},
		{"empty id", "fleet", map[string]string{"function": "heartbeat"}, `{"instanceId":""}`},
		{"not json", "fleet", map[string]string{"function": "heartbeat"}, `a`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := lp.NewEvent(tt.measurement, tt.tags, nil, tt.payload, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if id, err := ParseHeartbeat(m); err == nil {
				t.Errorf("expected error, got id %q", id)
			}
		})
	}
}

func TestEncodeRoster_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		providers []Provider
	}{
		{"empty", nil},
		{"providers", []Provider{
			{Type: ServiceBackend, Hostname: "mgmt01", State: "active", Meta: map[string]string{"port": "8080"}},
			{Type: ServiceMetricStore, Hostname: "mgmt02", State: "active"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := EncodeRoster("fritz", ServiceMetricCollector, tt.providers, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeRoster(data)
			if err != nil {
				t.Fatal(err)
			}
			want := tt.providers
			if want == nil {
				want = []Provider{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodeRoster_Invalid(t *testing.T) {
	hb, _ := EncodeHeartbeat(time.Now(), "a")
	if _, err := DecodeRoster(hb); err == nil {
		t.Error("expected error for a heartbeat message")
	}
	m, _ := lp.NewEvent(DiscoveryMeasurement, nil, nil, `{"not":"an array"}`, time.Now())
	if _, err := DecodeRoster([]byte(m.ToLineProtocol(nil))); err == nil {
		t.Error("expected error for a non-array payload")
	}
}

func TestDiscoverySubject_Buckets(t *testing.T) {
	tests := []struct {
		prefix, cluster string
		consumer        ServiceType
		want            string
	}{
		{"", "fritz", ServiceMetricCollector, "cc.fleet.discovery.fritz.ccmc"},
		{"", "", ServiceMetricStore, "cc.fleet.discovery.infra.ccms"},
		{"site.disc", "alex", ServiceSlurmAdapter, "site.disc.alex.ccsa"},
	}
	for _, tt := range tests {
		if got := DiscoverySubject(tt.prefix, Bucket(tt.cluster), tt.consumer); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestServiceType_Valid(t *testing.T) {
	for _, st := range ServiceTypes {
		if !st.Valid() {
			t.Errorf("%q should be valid", st)
		}
	}
	if ServiceType("xyz").Valid() {
		t.Error("unknown type should be invalid")
	}
}
