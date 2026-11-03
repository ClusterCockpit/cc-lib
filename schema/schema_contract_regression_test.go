// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding/json"
	"fmt"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"io"
	"strings"
	"testing"
)

func schemaContractCompile(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.LoadURL = func(url string) (io.ReadCloser, error) {
		if !strings.HasPrefix(url, "embedfs://") {
			return nil, fmt.Errorf("unexpected schema URL %s", url)
		}
		return schemaFiles.Open("schemas/" + strings.TrimPrefix(url, "embedfs://"))
	}
	result, err := compiler.Compile("embedfs://" + name)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func schemaContractMetric() map[string]any {
	return map[string]any{"unit": map[string]any{"base": "B"}, "timestep": 1, "series": []any{map[string]any{"hostname": "node001", "data": []any{1, 2, 3}, "statistics": map[string]any{"min": 1, "avg": 2, "max": 3}}}}
}
func schemaContractData() map[string]any {
	result := map[string]any{}
	for _, name := range []string{"cpu_user", "cpu_load", "mem_used", "flops_any", "mem_bw", "net_bw"} {
		result[name] = map[string]any{"node": schemaContractMetric()}
	}
	return result
}
func schemaContractJSON(t *testing.T, value any) any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
func TestSchemaContract_ScopeMapsRejectScalars(t *testing.T) {
	schema := schemaContractCompile(t, "job-data.schema.json")
	if err := schema.Validate(schemaContractJSON(t, schemaContractData())); err != nil {
		t.Fatal("valid control:", err)
	}
	for _, name := range []string{"cpu_user", "cpu_load", "flops_any", "mem_bw", "clock", "acc_utilization"} {
		for _, value := range []any{42, "wrong", true, []any{1}} {
			data := schemaContractData()
			data[name] = value
			if err := schema.Validate(schemaContractJSON(t, data)); err == nil {
				t.Errorf("%s accepts a non-map: %#v", name, value)
			}
		}
	}
	// Preserve the old nullable map behavior for fields whose type was implicit.
	data := schemaContractData()
	data["cpu_user"] = nil
	if err := schema.Validate(schemaContractJSON(t, data)); err != nil {
		t.Fatal("nullable map contract changed:", err)
	}
}

func TestSchemaContract_ConstructorStatisticsAreValid(t *testing.T) {
	schema := schemaContractCompile(t, "job-metric-data.schema.json")
	metric := JobMetric{Unit: Unit{Base: "B"}, Timestep: 1}
	for i := 0; i < 4; i++ {
		metric.Series = append(metric.Series, Series{Hostname: "node001", Data: []Float{1, 2}, Statistics: MetricStatistics{Min: 1, Avg: 1.5, Max: 2}})
	}
	metric.AddStatisticsSeries()
	if metric.StatisticsSeries == nil {
		t.Fatal("fixture failed to construct statistics")
	}
	if err := schema.Validate(schemaContractJSON(t, metric)); err != nil {
		t.Fatalf("constructor output rejected: %v", err)
	}
}

func TestSchemaContract_EachDataSampleHasNumericType(t *testing.T) {
	schema := schemaContractCompile(t, "job-metric-data.schema.json")
	for _, values := range []any{[]any{1, "wrong", true, map[string]any{}}, []any{1, true}} {
		metric := schemaContractMetric()
		metric["series"].([]any)[0].(map[string]any)["data"] = values
		if err := schema.Validate(schemaContractJSON(t, metric)); err == nil {
			t.Errorf("non-numeric data accepted: %#v", values)
		}
	}
	metric := schemaContractMetric()
	metric["series"].([]any)[0].(map[string]any)["data"] = []any{1, nil, 3}
	if err := schema.Validate(schemaContractJSON(t, metric)); err != nil {
		t.Fatal("missing-sample control rejected:", err)
	}
}

func TestSchemaContract_AdditionalPercentilesHaveTypedValues(t *testing.T) {
	schema := schemaContractCompile(t, "job-metric-data.schema.json")
	for _, bad := range []map[string]any{{"37": "wrong"}, {"37": true}, {"unexpected": []any{1, 2, 3}}} {
		metric := schemaContractMetric()
		metric["statisticsSeries"] = map[string]any{"percentiles": bad}
		if err := schema.Validate(schemaContractJSON(t, metric)); err == nil {
			t.Errorf("invalid percentile map accepted: %#v", bad)
		}
	}
	metric := schemaContractMetric()
	metric["statisticsSeries"] = map[string]any{"percentiles": map[string]any{"37": []any{1, nil, 3}}}
	if err := schema.Validate(schemaContractJSON(t, metric)); err != nil {
		t.Fatal("supported percentile rejected:", err)
	}
}
