// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"testing"
)

func TestAddNodeScope(t *testing.T) {
	// Two hosts with core-level data of different lengths.
	// host1 has 2 cores with 3 data points each.
	// host2 has 2 cores with 5 data points each.
	jd := JobData{
		"flops_any": {
			MetricScopeCore: &JobMetric{
				Unit:     Unit{Base: "F/s"},
				Timestep: 10,
				Series: []Series{
					{Hostname: "host1", Data: []Float{1, 2, 3}, Statistics: MetricStatistics{Min: 1, Avg: 2, Max: 3}},
					{Hostname: "host1", Data: []Float{4, 5, 6}, Statistics: MetricStatistics{Min: 4, Avg: 5, Max: 6}},
					{Hostname: "host2", Data: []Float{10, 20, 30, 40, 50}, Statistics: MetricStatistics{Min: 10, Avg: 30, Max: 50}},
					{Hostname: "host2", Data: []Float{100, 200, 300, 400, 500}, Statistics: MetricStatistics{Min: 100, Avg: 300, Max: 500}},
				},
			},
		},
	}

	ok := jd.AddNodeScope("flops_any")
	if !ok {
		t.Fatal("AddNodeScope returned false")
	}

	nodeMetric, exists := jd["flops_any"][MetricScopeNode]
	if !exists {
		t.Fatal("node scope not created")
	}

	if len(nodeMetric.Series) != 2 {
		t.Fatalf("expected 2 node series, got %d", len(nodeMetric.Series))
	}

	// Build a map for deterministic checking (range over map is random order).
	byHost := make(map[string]Series)
	for _, s := range nodeMetric.Series {
		byHost[s.Hostname] = s
	}

	// host1: sum of cores = [1+4, 2+5, 3+6] = [5, 7, 9], length 3
	h1 := byHost["host1"]
	if len(h1.Data) != 3 {
		t.Fatalf("host1: expected 3 data points, got %d", len(h1.Data))
	}
	expectH1 := []Float{5, 7, 9}
	for i, v := range expectH1 {
		if h1.Data[i] != v {
			t.Errorf("host1 data[%d]: expected %v, got %v", i, v, h1.Data[i])
		}
	}

	// host2: sum of cores = [110, 220, 330, 440, 550], length 5
	h2 := byHost["host2"]
	if len(h2.Data) != 5 {
		t.Fatalf("host2: expected 5 data points, got %d", len(h2.Data))
	}
	expectH2 := []Float{110, 220, 330, 440, 550}
	for i, v := range expectH2 {
		if h2.Data[i] != v {
			t.Errorf("host2 data[%d]: expected %v, got %v", i, v, h2.Data[i])
		}
	}
}

func TestAddNodeScopeUnevenCores(t *testing.T) {
	// Same host, cores with different data lengths.
	jd := JobData{
		"mem_bw": {
			MetricScopeCore: &JobMetric{
				Unit:     Unit{Base: "B/s"},
				Timestep: 10,
				Series: []Series{
					{Hostname: "node1", Data: []Float{1, 2, 3}, Statistics: MetricStatistics{Min: 1, Avg: 2, Max: 3}},
					{Hostname: "node1", Data: []Float{10, 20, 30, 40, 50}, Statistics: MetricStatistics{Min: 10, Avg: 30, Max: 50}},
				},
			},
		},
	}

	ok := jd.AddNodeScope("mem_bw")
	if !ok {
		t.Fatal("AddNodeScope returned false")
	}

	nodeMetric := jd["mem_bw"][MetricScopeNode]
	if len(nodeMetric.Series) != 1 {
		t.Fatalf("expected 1 node series, got %d", len(nodeMetric.Series))
	}

	s := nodeMetric.Series[0]
	// n=5 (max length), m=3 (min length)
	// data[0..2] = sum, data[3..4] = NaN
	if len(s.Data) != 5 {
		t.Fatalf("expected 5 data points, got %d", len(s.Data))
	}

	expect := []Float{11, 22, 33}
	for i, v := range expect {
		if s.Data[i] != v {
			t.Errorf("data[%d]: expected %v, got %v", i, v, s.Data[i])
		}
	}

	// Remaining should be NaN
	for i := 3; i < 5; i++ {
		if !s.Data[i].IsNaN() {
			t.Errorf("data[%d]: expected NaN, got %v", i, s.Data[i])
		}
	}
}
