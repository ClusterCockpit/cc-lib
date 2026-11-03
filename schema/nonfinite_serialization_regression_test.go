// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func TestNonfiniteJSON_RejectsUnrepresentableNumbers(t *testing.T) {
	for _, value := range []float64{math.Inf(1), math.Inf(-1)} {
		cases := []json.Marshaler{Float(value), FloatArray{Float(value)}, &Series{Hostname: "node001", Data: []Float{Float(value)}}, &Series{Hostname: "node001", Statistics: MetricStatistics{Min: value}}}
		for _, item := range cases {
			if data, err := item.MarshalJSON(); err == nil {
				t.Errorf("%T emitted unrepresentable JSON %q without error", item, data)
			}
		}
	}
	series := Series{Hostname: "node001", Statistics: MetricStatistics{Avg: math.NaN()}}
	if data, err := series.MarshalJSON(); err == nil {
		t.Errorf("nonfinite statistics emitted %q without error", data)
	}
}
func TestNonfiniteJSON_PreservesMissingContract(t *testing.T) {
	for _, item := range []json.Marshaler{NaN, FloatArray{NaN}, &Series{Hostname: "node001", Data: []Float{NaN}}} {
		data, err := item.MarshalJSON()
		if err != nil || !json.Valid(data) {
			t.Errorf("missing contract rejected: %q %v", data, err)
		}
	}
}
func TestNonfiniteGraphQL_UsesNullableScalar(t *testing.T) {
	for _, value := range []Float{NaN, Float(math.Inf(1)), Float(math.Inf(-1))} {
		var output bytes.Buffer
		value.MarshalGQL(&output)
		if output.String() != "null" {
			t.Errorf("nonfinite GraphQL output %q", output.String())
		}
	}
}
