// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"math"
	"testing"
)

func TestStatistics_MedianOmitsMissing(t *testing.T) {
	jm := JobMetric{Series: []Series{{Data: []Float{NaN}}, {Data: []Float{10}}, {Data: []Float{20}}, {Data: []Float{30}}}}
	jm.AddStatisticsSeries()
	if got := jm.StatisticsSeries.Median[0]; got != 20 {
		t.Fatalf("median=%v, want20", got)
	}
	if math.IsNaN(float64(jm.StatisticsSeries.Min[0])) {
		t.Fatal("finite minimum lost")
	}
}

func TestStatistics_ThreeSeriesPercentilesDoNotPanic(t *testing.T) {
	jm := JobMetric{Series: []Series{{Data: []Float{1}}, {Data: []Float{2}}, {Data: []Float{3}}}}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("three series caused panic: %v", p)
		}
	}()
	if jm.AddPercentiles([]int{50}) {
		t.Fatal("absent statistics unexpectedly created for three series")
	}
}

func TestStatistics_PercentilesOmitMissing(t *testing.T) {
	jm := JobMetric{}
	for _, value := range []Float{NaN, NaN, 10, 20, 30} {
		jm.Series = append(jm.Series, Series{Data: []Float{value, NaN}})
	}
	if !jm.AddPercentiles([]int{50}) {
		t.Fatal("percentiles unavailable")
	}
	values := jm.StatisticsSeries.Percentiles[50]
	if values[0] != 20 {
		t.Errorf("P50=%v, want20", values[0])
	}
	if !values[1].IsNaN() {
		t.Errorf("missing-only percentile=%v", values[1])
	}
}

func TestStatistics_UnevenSeriesRetainAvailableSamples(t *testing.T) {
	jm := JobMetric{Series: []Series{{Data: []Float{1}}, {Data: []Float{10, 20}}, {Data: []Float{20, 30}}, {Data: []Float{30, 40}}}}
	jm.AddStatisticsSeries()
	stats := jm.StatisticsSeries
	if len(stats.Median) != 2 || stats.Median[1] != 30 || stats.Min[1] != 20 || stats.Max[1] != 40 {
		t.Fatalf("available tail samples discarded: %+v", stats)
	}
}
