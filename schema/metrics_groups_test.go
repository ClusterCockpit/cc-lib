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

// jobDataFixture is a schema-valid job-data document. It contains every required
// top-level metric, a top-level metric named "open" that collides with a
// filesystem sub-metric of the same name, and a two-instance filesystems array.
func jobDataFixture() string {
	m := nodeMetricJSON
	return `{` +
		`"cpu_user":` + m + `,` +
		`"cpu_load":` + m + `,` +
		`"mem_used":` + m + `,` +
		`"flops_any":` + m + `,` +
		`"mem_bw":` + m + `,` +
		`"net_bw":` + m + `,` +
		`"open":` + m + `,` +
		`"filesystems":[` +
		`{"name":"home","type":"nfs","read_bw":` + m + `,"write_bw":` + m + `,"open":` + m + `},` +
		`{"name":"scratch","type":"lustre","read_bw":` + m + `,"write_bw":` + m + `}` +
		`]}`
}

func TestJobData_GroupsRoundTrip(t *testing.T) {
	fixture := jobDataFixture()

	if err := Validate(Data, strings.NewReader(fixture)); err != nil {
		t.Fatalf("fixture does not validate against job-data schema: %v", err)
	}

	var jd JobData
	if err := json.Unmarshal([]byte(fixture), &jd); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	// Flat metric and the colliding top-level "open" survive.
	if _, ok := jd.Metrics["flops_any"]; !ok {
		t.Error("flat metric flops_any missing")
	}
	if _, ok := jd.Metrics["open"]; !ok {
		t.Error("top-level metric open missing (collision)")
	}
	if _, ok := jd.Metrics["filesystems"]; ok {
		t.Error("filesystems must not land in flat Metrics")
	}

	// Group parsed with instance identity and its own metrics (incl. its own "open").
	if len(jd.Groups) != 1 || jd.Groups[0].Key != "filesystems" {
		t.Fatalf("expected one filesystems group, got %+v", jd.Groups)
	}
	insts := jd.Groups[0].Instances
	if len(insts) != 2 {
		t.Fatalf("expected 2 filesystem instances, got %d", len(insts))
	}
	if insts[0].Name != "home" || insts[0].Type != "nfs" {
		t.Errorf("instance[0] identity wrong: %+v", insts[0])
	}
	if insts[1].Name != "scratch" || insts[1].Type != "lustre" {
		t.Errorf("instance[1] identity wrong: %+v", insts[1])
	}
	if _, ok := insts[0].Metrics["open"]; !ok {
		t.Error("filesystem sub-metric open missing (collision with top-level open)")
	}
	if _, ok := insts[0].Metrics["read_bw"]; !ok {
		t.Error("filesystem sub-metric read_bw missing")
	}

	// Marshal back -> still schema-valid, and filesystems is an array.
	out, err := json.Marshal(jd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := Validate(Data, bytes.NewReader(out)); err != nil {
		t.Fatalf("re-marshalled job data does not validate: %v", err)
	}
	if !bytes.Contains(out, []byte(`"filesystems":[`)) {
		t.Errorf("marshalled output does not contain filesystems array: %s", out)
	}

	// Re-unmarshal and compare structurally (instance order preserved).
	var jd2 JobData
	if err := json.Unmarshal(out, &jd2); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if !reflect.DeepEqual(jd, jd2) {
		t.Errorf("round-trip mismatch:\n first: %+v\nsecond: %+v", jd, jd2)
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

func TestJobData_SizeIncludesGroups(t *testing.T) {
	mk := func() ScopedMetrics {
		return ScopedMetrics{
			MetricScopeNode: &JobMetric{
				Unit:     Unit{Base: "B/s"},
				Timestep: 60,
				Series:   []Series{{Hostname: "h1", Data: []Float{1, 2, 3, 4, 5}}},
			},
		}
	}

	base := JobData{Metrics: map[string]ScopedMetrics{"read_bw": mk()}}
	withGroup := JobData{Metrics: map[string]ScopedMetrics{"read_bw": mk()}}
	withGroup.AddGroupInstance("filesystems", "home", "nfs", map[string]ScopedMetrics{"read_bw": mk()})

	if withGroup.Size() <= base.Size() {
		t.Errorf("Size() must account for group metrics: base=%d withGroup=%d", base.Size(), withGroup.Size())
	}
}

func TestJobStatisticsSet_RoundTrip(t *testing.T) {
	set := JobStatisticsSet{
		Metrics: map[string]JobStatistics{
			"cpu_load": {Unit: Unit{Base: ""}, Avg: 2, Min: 1, Max: 3},
			"mem_bw":   {Unit: Unit{Base: "B/s", Prefix: "G"}, Avg: 20, Min: 10, Max: 30},
		},
	}
	set.AddGroupInstance("filesystems", "home", "nfs", map[string]JobStatistics{
		"read_bw":  {Unit: Unit{Base: "B/s"}, Avg: 5, Min: 1, Max: 9},
		"write_bw": {Unit: Unit{Base: "B/s"}, Avg: 6, Min: 2, Max: 8},
	})

	out, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !bytes.Contains(out, []byte(`"filesystems":[`)) {
		t.Errorf("statistics output missing filesystems array: %s", out)
	}
	if !bytes.Contains(out, []byte(`"cpu_load"`)) {
		t.Errorf("statistics output missing flat metric: %s", out)
	}

	var got JobStatisticsSet
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(set, got) {
		t.Errorf("round-trip mismatch:\n first: %+v\nsecond: %+v", set, got)
	}
}
