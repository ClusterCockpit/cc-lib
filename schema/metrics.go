// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

import (
	"fmt"
	"io"
	"math"
	"sort"
	"unsafe"

	"github.com/ClusterCockpit/cc-lib/v2/util"
)

// JobData maps metric names to their data organized by scope.
// Structure: map[metricName]map[scope]*JobMetric
//
// For example: jobData["cpu_load"][MetricScopeNode] contains node-level CPU load data.
// This structure allows efficient lookup of metrics at different hierarchical levels.
type JobData map[string]map[MetricScope]*JobMetric

// ScopedJobStats maps metric names to statistical summaries organized by scope.
// Structure: map[metricName]map[scope][]*ScopedStats
//
// Used to store pre-computed statistics without the full time series data,
// reducing memory footprint when only aggregated values are needed.
type ScopedJobStats map[string]map[MetricScope][]*ScopedStats

// JobMetric contains time series data and statistics for a single metric.
//
// The Series field holds time series data from individual nodes/hardware components,
// while StatisticsSeries provides aggregated statistics across all series over time.
type JobMetric struct {
	StatisticsSeries *StatsSeries `json:"statisticsSeries,omitempty"` // Aggregated statistics over time
	Unit             Unit         `json:"unit"`                       // Unit of measurement
	Series           []Series     `json:"series"`                     // Individual time series data
	Timestep         int          `json:"timestep"`                   // Sampling interval in seconds
}

// Series represents a single time series of metric measurements.
//
// Each series corresponds to one source (e.g., one node, one core) identified by Hostname and optional ID.
// The Data field contains the time-ordered measurements, and Statistics provides min/avg/max summaries.
type Series struct {
	Id         *string          `json:"id,omitempty"` // Optional ID (e.g., core ID, GPU ID)
	Hostname   string           `json:"hostname"`     // Source hostname
	Data       []Float          `json:"data"`         // Time series measurements
	Statistics MetricStatistics `json:"statistics"`   // Statistical summary (min/avg/max)
}

// ScopedStats contains statistical summaries for a specific scope (e.g., one node, one socket).
// Used when full time series data isn't needed, only the aggregated statistics.
type ScopedStats struct {
	Hostname string            `json:"hostname"`     // Source hostname
	Id       *string           `json:"id,omitempty"` // Optional scope ID
	Data     *MetricStatistics `json:"data"`         // Statistical summary
}

// MetricStatistics holds statistical summary values for metric data.
// Provides the common statistical aggregations used throughout ClusterCockpit.
type MetricStatistics struct {
	Avg float64 `json:"avg"` // Average/mean value
	Min float64 `json:"min"` // Minimum value
	Max float64 `json:"max"` // Maximum value
}

// StatsSeries contains aggregated statistics across multiple time series over time.
//
// Instead of storing individual series, this provides statistical summaries at each time step.
// For example, at time t, Mean[t] is the average value across all series at that time.
// Percentiles provides specified percentile values at each time step.
type StatsSeries struct {
	Percentiles map[int][]Float `json:"percentiles,omitempty"` // Percentile values over time (e.g., 10th, 50th, 90th)
	Mean        []Float         `json:"mean"`                  // Mean values over time
	Median      []Float         `json:"median"`                // Median values over time
	Min         []Float         `json:"min"`                   // Minimum values over time
	Max         []Float         `json:"max"`                   // Maximum values over time
}

// MetricScope defines the hierarchical level at which a metric is measured.
//
// Scopes form a hierarchy from coarse-grained (node) to fine-grained (hwthread/accelerator):
//
//	node > socket > memoryDomain > core > hwthread
//	accelerator is a special scope at the same level as hwthread
//
// The scopePrecedence map defines numeric ordering for scope comparisons,
// which is used when aggregating metrics across different scopes.
type MetricScope string

const (
	MetricScopeInvalid MetricScope = "invalid_scope"

	MetricScopeNode         MetricScope = "node"
	MetricScopeSocket       MetricScope = "socket"
	MetricScopeMemoryDomain MetricScope = "memoryDomain"
	MetricScopeCore         MetricScope = "core"
	MetricScopeHWThread     MetricScope = "hwthread"

	// TODO: Add Job and Application scopes

	MetricScopeAccelerator MetricScope = "accelerator"
)

var metricScopeGranularity map[MetricScope]int = map[MetricScope]int{
	MetricScopeNode:         10,
	MetricScopeSocket:       5,
	MetricScopeMemoryDomain: 4,
	MetricScopeCore:         3,
	MetricScopeHWThread:     2,
	/* Special-Case Accelerator
	 * -> No conversion possible if native scope is HWTHREAD
	 * -> Therefore needs to be less than HWTREAD, else max() would return unhandled case
	 * -> If nativeScope is accelerator, accelerator metrics return correctly
	 */
	MetricScopeAccelerator: 1,

	MetricScopeInvalid: -1,
}

func (e *MetricScope) LT(other MetricScope) bool {
	a := metricScopeGranularity[*e]
	b := metricScopeGranularity[other]
	return a < b
}

func (e *MetricScope) LTE(other MetricScope) bool {
	a := metricScopeGranularity[*e]
	b := metricScopeGranularity[other]
	return a <= b
}

func (e *MetricScope) Max(other MetricScope) MetricScope {
	a := metricScopeGranularity[*e]
	b := metricScopeGranularity[other]
	if a > b {
		return *e
	}
	return other
}

func (e *MetricScope) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("SCHEMA/METRICS > enums must be strings")
	}

	*e = MetricScope(str)
	if !e.Valid() {
		return fmt.Errorf("SCHEMA/METRICS > %s is not a valid MetricScope", str)
	}
	return nil
}

func (e MetricScope) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, "\"%s\"", e)
}

func (e MetricScope) Valid() bool {
	gran, ok := metricScopeGranularity[e]
	return ok && gran > 0
}

func (jd *JobData) Size() int {
	n := 128
	for _, scopes := range *jd {
		for _, metric := range scopes {
			if metric.StatisticsSeries != nil {
				n += len(metric.StatisticsSeries.Max)
				n += len(metric.StatisticsSeries.Mean)
				n += len(metric.StatisticsSeries.Median)
				n += len(metric.StatisticsSeries.Min)
			}

			for _, series := range metric.Series {
				n += len(series.Data)
			}
		}
	}
	return n * int(unsafe.Sizeof(Float(0)))
}

const smooth bool = false

func (jm *JobMetric) AddStatisticsSeries() {
	if jm.StatisticsSeries != nil || len(jm.Series) < 4 {
		return
	}

	n, m := 0, len(jm.Series[0].Data)
	for _, series := range jm.Series {
		if len(series.Data) > n {
			n = len(series.Data)
		}
		if len(series.Data) < m {
			m = len(series.Data)
		}
	}

	// mean := make([]Float, n)
	min, median, max := make([]Float, n), make([]Float, n), make([]Float, n)
	i := 0
	for ; i < m; i++ {
		seriesCount := len(jm.Series)
		// ssum := 0.0
		smin, smed, smax := math.MaxFloat32, make([]float64, seriesCount), -math.MaxFloat32
		notnan := 0
		for j := range seriesCount {
			x := float64(jm.Series[j].Data[i])
			if math.IsNaN(x) {
				continue
			}

			notnan += 1
			// ssum += x
			smed[j] = x
			smin = math.Min(smin, x)
			smax = math.Max(smax, x)
		}

		if notnan < 3 {
			min[i] = NaN
			// mean[i] = NaN
			median[i] = NaN
			max[i] = NaN
		} else {
			min[i] = Float(smin)
			// mean[i] = Float(ssum / float64(notnan))
			max[i] = Float(smax)

			medianRaw, err := util.Median(smed)
			if err != nil {
				median[i] = NaN
			} else {
				median[i] = Float(medianRaw)
			}
		}
	}

	for ; i < n; i++ {
		min[i] = NaN
		// mean[i] = NaN
		median[i] = NaN
		max[i] = NaN
	}

	if smooth {
		for i := 2; i < len(median)-2; i++ {
			if min[i].IsNaN() {
				continue
			}

			min[i] = (min[i-2] + min[i-1] + min[i] + min[i+1] + min[i+2]) / 5
			max[i] = (max[i-2] + max[i-1] + max[i] + max[i+1] + max[i+2]) / 5
			// mean[i] = (mean[i-2] + mean[i-1] + mean[i] + mean[i+1] + mean[i+2]) / 5
			// Reduce Median further
			smoothRaw := []float64{float64(median[i-2]), float64(median[i-1]), float64(median[i]), float64(median[i+1]), float64(median[i+2])}
			smoothMedian, err := util.Median(smoothRaw)
			if err != nil {
				median[i] = NaN
			} else {
				median[i] = Float(smoothMedian)
			}
		}
	}

	jm.StatisticsSeries = &StatsSeries{Median: median, Min: min, Max: max} // Mean: mean
}

func (jd *JobData) AddNodeScope(metric string) bool {
	scopes, ok := (*jd)[metric]
	if !ok {
		return false
	}

	maxScope := MetricScopeInvalid
	for scope := range scopes {
		maxScope = maxScope.Max(scope)
	}

	if maxScope == MetricScopeInvalid || maxScope == MetricScopeNode {
		return false
	}

	jm := scopes[maxScope]
	hosts := make(map[string][]Series, 32)
	for _, series := range jm.Series {
		hosts[series.Hostname] = append(hosts[series.Hostname], series)
	}

	nodeJm := &JobMetric{
		Unit:     jm.Unit,
		Timestep: jm.Timestep,
		Series:   make([]Series, 0, len(hosts)),
	}
	for hostname, series := range hosts {
		min, sum, max := math.MaxFloat32, 0.0, -math.MaxFloat32
		for _, series := range series {
			sum += series.Statistics.Avg
			min = math.Min(min, series.Statistics.Min)
			max = math.Max(max, series.Statistics.Max)
		}

		n, m := 0, len(jm.Series[0].Data)
		for _, series := range jm.Series {
			if len(series.Data) > n {
				n = len(series.Data)
			}
			if len(series.Data) < m {
				m = len(series.Data)
			}
		}

		i, data := 0, make([]Float, len(series[0].Data))
		for ; i < m; i++ {
			x := Float(0.0)
			for _, series := range jm.Series {
				x += series.Data[i]
			}
			data[i] = x
		}

		for ; i < n; i++ {
			data[i] = NaN
		}

		nodeJm.Series = append(nodeJm.Series, Series{
			Hostname:   hostname,
			Statistics: MetricStatistics{Min: min, Avg: sum / float64(len(series)), Max: max},
			Data:       data,
		})
	}

	scopes[MetricScopeNode] = nodeJm
	return true
}

func (jd *JobData) RoundMetricStats() {
	// TODO: Make Digit-Precision Configurable? (Currently: Fixed to 2 Digits)
	for _, scopes := range *jd {
		for _, jm := range scopes {
			for index := range jm.Series {
				jm.Series[index].Statistics = MetricStatistics{
					Avg: (math.Round(jm.Series[index].Statistics.Avg*100) / 100),
					Min: (math.Round(jm.Series[index].Statistics.Min*100) / 100),
					Max: (math.Round(jm.Series[index].Statistics.Max*100) / 100),
				}
			}
		}
	}
}

func (sjs *ScopedJobStats) RoundScopedMetricStats() {
	// TODO: Make Digit-Precision Configurable? (Currently: Fixed to 2 Digits)
	for _, scopes := range *sjs {
		for _, stats := range scopes {
			for index := range stats {
				roundedStats := MetricStatistics{
					Avg: (math.Round(stats[index].Data.Avg*100) / 100),
					Min: (math.Round(stats[index].Data.Min*100) / 100),
					Max: (math.Round(stats[index].Data.Max*100) / 100),
				}
				stats[index].Data = &roundedStats
			}
		}
	}
}

func (jm *JobMetric) AddPercentiles(ps []int) bool {
	if jm.StatisticsSeries == nil {
		jm.AddStatisticsSeries()
	}

	if len(jm.Series) < 3 {
		return false
	}

	if jm.StatisticsSeries.Percentiles == nil {
		jm.StatisticsSeries.Percentiles = make(map[int][]Float, len(ps))
	}

	n := 0
	for _, series := range jm.Series {
		if len(series.Data) > n {
			n = len(series.Data)
		}
	}

	data := make([][]float64, n)
	for i := range n {
		vals := make([]float64, 0, len(jm.Series))
		for _, series := range jm.Series {
			if i < len(series.Data) {
				vals = append(vals, float64(series.Data[i]))
			}
		}

		sort.Float64s(vals)
		data[i] = vals
	}

	for _, p := range ps {
		if p < 1 || p > 99 {
			panic("SCHEMA/METRICS > invalid percentile")
		}

		if _, ok := jm.StatisticsSeries.Percentiles[p]; ok {
			continue
		}

		percentiles := make([]Float, n)
		for i := range n {
			sorted := data[i]
			percentiles[i] = Float(sorted[(len(sorted)*p)/100])
		}

		jm.StatisticsSeries.Percentiles[p] = percentiles
	}

	return true
}
