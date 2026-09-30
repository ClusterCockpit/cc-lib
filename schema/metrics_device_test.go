// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// one scope object of job-metric-data, reused across the fixtures below.
const nodeMetricJSON = `{"node":{"unit":{"base":"B/s"},"timestep":60,"series":[{"hostname":"h1","statistics":{"avg":2,"min":1,"max":3},"data":[1,2,3]}]}}`

// fsMetricJSON is a filesystem metric carrying both its node total and one
// series per mount point, identified by the series id.
const fsMetricJSON = `{` +
	`"node":{"unit":{"base":"B/s"},"timestep":60,"series":[` +
	`{"hostname":"h1","statistics":{"avg":6,"min":3,"max":9},"data":[3,6,9]}]},` +
	`"filesystem":{"unit":{"base":"B/s"},"timestep":60,"series":[` +
	`{"hostname":"h1","id":"/home","statistics":{"avg":2,"min":1,"max":3},"data":[1,2,3]},` +
	`{"hostname":"h1","id":"/scratch","statistics":{"avg":4,"min":2,"max":6},"data":[2,4,6]}]}}`

// jobDataFixture is a schema-valid job-data document with every required
// metric and one filesystem metric.
func jobDataFixture() string {
	m := nodeMetricJSON
	return `{` +
		`"cpu_user":` + m + `,` +
		`"cpu_load":` + m + `,` +
		`"mem_used":` + m + `,` +
		`"flops_any":` + m + `,` +
		`"mem_bw":` + m + `,` +
		`"net_bw":` + m + `,` +
		`"fs_read_bw":` + fsMetricJSON +
		`}`
}

func TestJobData_DeviceScopeRoundTrip(t *testing.T) {
	fixture := jobDataFixture()

	if err := Validate(Data, strings.NewReader(fixture)); err != nil {
		t.Fatalf("fixture does not validate against job-data schema: %v", err)
	}

	var jd JobData
	if err := json.Unmarshal([]byte(fixture), &jd); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	fs, ok := jd.Metrics["fs_read_bw"][MetricScopeFilesystem]
	if !ok {
		t.Fatal("fs_read_bw has no filesystem scope")
	}
	if len(fs.Series) != 2 {
		t.Fatalf("expected 2 filesystem series, got %d", len(fs.Series))
	}
	for i, want := range []string{"/home", "/scratch"} {
		if fs.Series[i].ID == nil || *fs.Series[i].ID != want {
			t.Errorf("series[%d] id = %v, want %q", i, fs.Series[i].ID, want)
		}
	}
	if _, ok := jd.Metrics["fs_read_bw"][MetricScopeNode]; !ok {
		t.Error("fs_read_bw node total missing")
	}

	out, err := json.Marshal(jd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := Validate(Data, bytes.NewReader(out)); err != nil {
		t.Fatalf("re-marshalled job data does not validate: %v", err)
	}

	var jd2 JobData
	if err := json.Unmarshal(out, &jd2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if !reflect.DeepEqual(jd, jd2) {
		t.Errorf("round-trip mismatch:\n first: %+v\nsecond: %+v", jd, jd2)
	}
}

func TestJobData_UnmarshalRejectsFilesystemsArray(t *testing.T) {
	legacy := `{"filesystems":[{"name":"home","type":"nfs","read_bw":` + nodeMetricJSON + `}]}`

	var jd JobData
	if err := json.Unmarshal([]byte(legacy), &jd); err == nil {
		t.Errorf("expected an error for the removed filesystems array, got %+v", jd)
	}
}

func TestJobData_NaNMarshalsAsNull(t *testing.T) {
	id := "0"
	jd := JobData{
		Metrics: map[string]ScopedMetrics{
			"flops_any": {
				MetricScopeNode: &JobMetric{
					Unit:     Unit{Base: "F/s"},
					Timestep: 60,
					Series: []Series{
						{Hostname: "h1", ID: &id, Data: []Float{1, NaN, 3}, Statistics: MetricStatistics{Min: 1, Avg: 2, Max: 3}},
					},
				},
			},
		},
	}

	out, err := json.Marshal(jd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(out, []byte("null")) {
		t.Errorf("NaN data point not rendered as null: %s", out)
	}
}

func TestScopedJobStats_RoundTrip(t *testing.T) {
	home, scratch := "/home", "/scratch"
	sjs := ScopedJobStats{
		Metrics: map[string]ScopedMetricStats{
			"fs_read_bw": {
				MetricScopeFilesystem: {
					{Hostname: "h1", ID: &home, Data: &MetricStatistics{Avg: 2, Min: 1, Max: 3}},
					{Hostname: "h1", ID: &scratch, Data: &MetricStatistics{Avg: 4, Min: 2, Max: 6}},
				},
			},
		},
	}

	out, err := json.Marshal(sjs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.HasPrefix(out, []byte(`{"fs_read_bw":{"filesystem":[`)) {
		t.Errorf("unexpected layout: %s", out)
	}

	var got ScopedJobStats
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(sjs, got) {
		t.Errorf("round-trip mismatch:\n first: %+v\nsecond: %+v", sjs, got)
	}
}

func TestJobStatisticsSet_RoundTrip(t *testing.T) {
	set := JobStatisticsSet{
		Metrics: map[string]JobStatistics{
			"cpu_load":   {Unit: Unit{Base: ""}, Avg: 2, Min: 1, Max: 3},
			"fs_read_bw": {Unit: Unit{Base: "B/s", Prefix: "G"}, Avg: 20, Min: 10, Max: 30},
		},
	}

	out, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(out, []byte(`"fs_read_bw":{"unit":{"base":"B/s","prefix":"G"}`)) {
		t.Errorf("statistics output has unexpected layout: %s", out)
	}

	var got JobStatisticsSet
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(set, got) {
		t.Errorf("round-trip mismatch:\n first: %+v\nsecond: %+v", set, got)
	}
}

func TestMetricScope_IsDevice(t *testing.T) {
	tests := []struct {
		scope MetricScope
		want  bool
	}{
		{MetricScopeNode, false},
		{MetricScopeSocket, false},
		{MetricScopeMemoryDomain, false},
		{MetricScopeCore, false},
		{MetricScopeHWThread, false},
		{MetricScopeAccelerator, true},
		{MetricScopeFilesystem, true},
		{MetricScopeNetwork, true},
		{MetricScopeInvalid, false},
	}
	for _, tt := range tests {
		if got := tt.scope.IsDevice(); got != tt.want {
			t.Errorf("%s.IsDevice() = %v, want %v", tt.scope, got, tt.want)
		}
	}
}

func TestMetricScope_DeviceScopesValid(t *testing.T) {
	for _, s := range []MetricScope{MetricScopeFilesystem, MetricScopeNetwork} {
		if !s.Valid() {
			t.Errorf("%s must be a valid scope", s)
		}
		var parsed MetricScope
		if err := parsed.UnmarshalGQL(string(s)); err != nil || parsed != s {
			t.Errorf("UnmarshalGQL(%q) = %q, %v", s, parsed, err)
		}
	}
}

// TestMetricScope_MaxWithDeviceScopes pins the Max() results that query
// builders rely on: a device scope aggregates only to node, and is never
// replaced by a CPU scope finer than node.
func TestMetricScope_MaxWithDeviceScopes(t *testing.T) {
	tests := []struct {
		native, requested, want MetricScope
	}{
		{MetricScopeFilesystem, MetricScopeFilesystem, MetricScopeFilesystem},
		{MetricScopeFilesystem, MetricScopeNode, MetricScopeNode},
		{MetricScopeFilesystem, MetricScopeCore, MetricScopeCore},
		{MetricScopeNetwork, MetricScopeNetwork, MetricScopeNetwork},
		{MetricScopeNetwork, MetricScopeNode, MetricScopeNode},
		{MetricScopeHWThread, MetricScopeFilesystem, MetricScopeHWThread},
		{MetricScopeNode, MetricScopeNetwork, MetricScopeNode},
		// Equal granularity: Max returns the requested scope, which differs
		// from the native one and must be rejected by the caller.
		{MetricScopeFilesystem, MetricScopeAccelerator, MetricScopeAccelerator},
	}
	for _, tt := range tests {
		if got := tt.native.Max(tt.requested); got != tt.want {
			t.Errorf("%s.Max(%s) = %s, want %s", tt.native, tt.requested, got, tt.want)
		}
	}
}

func TestAddNodeScope_Filesystem(t *testing.T) {
	var jd JobData
	if err := json.Unmarshal([]byte(`{"fs_read_bw":`+fsMetricJSON+`}`), &jd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	delete(jd.Metrics["fs_read_bw"], MetricScopeNode)

	if !jd.AddNodeScope("fs_read_bw") {
		t.Fatal("AddNodeScope returned false")
	}
	node := jd.Metrics["fs_read_bw"][MetricScopeNode]
	if len(node.Series) != 1 {
		t.Fatalf("expected one node series, got %d", len(node.Series))
	}
	want := []Float{3, 6, 9}
	if !reflect.DeepEqual(node.Series[0].Data, want) {
		t.Errorf("node total = %v, want %v", node.Series[0].Data, want)
	}
}

func TestTopology_GetDeviceIDs(t *testing.T) {
	topo := Topology{
		Accelerators: []*Accelerator{{ID: "0", Type: "Nvidia GPU", Model: "A100"}},
		Filesystems:  []*Filesystem{{ID: "/home", Type: "nfs"}, {ID: "/scratch", Type: "lustre"}},
		Networks:     []*Network{{ID: "ib0", Type: "infiniband"}},
	}

	tests := []struct {
		scope MetricScope
		want  []string
	}{
		{MetricScopeAccelerator, []string{"0"}},
		{MetricScopeFilesystem, []string{"/home", "/scratch"}},
		{MetricScopeNetwork, []string{"ib0"}},
		{MetricScopeCore, nil},
		{MetricScopeNode, nil},
	}
	for _, tt := range tests {
		if got := topo.GetDeviceIDs(tt.scope); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("GetDeviceIDs(%s) = %v, want %v", tt.scope, got, tt.want)
		}
	}

	var empty Topology
	if got := empty.GetFilesystemIDs(); len(got) != 0 {
		t.Errorf("GetFilesystemIDs on empty topology = %v, want empty", got)
	}
}
