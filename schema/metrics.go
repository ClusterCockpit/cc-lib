// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"unsafe"

	"github.com/ClusterCockpit/cc-lib/v2/util"
)

// ScopedMetrics maps a hierarchical scope (node/socket/core/...) to its metric
// data. It is the value stored for a single metric name.
type ScopedMetrics map[MetricScope]*JobMetric

// MetricGroupInstance is one named element of an array-valued metric group,
// such as a single filesystem (and, in the future, a single network
// interconnect). Alongside its identity (Name/Type) it carries its own set of
// scoped metrics keyed by metric name (e.g. "read_bw", "write_bw").
//
// In the job-archive JSON, per-instance metrics are node-scoped; the scope map
// is kept general so finer scopes remain representable without type changes.
type MetricGroupInstance struct {
	Name    string                   `json:"name"`
	Type    string                   `json:"type"`
	Metrics map[string]ScopedMetrics `json:"-"` // flattened onto the instance object by MarshalJSON
}

// MetricGroup is an array-valued group of named instances, identified by its
// top-level JSON key (e.g. "filesystems"). Instance order is preserved because
// the schema array order is meaningful for rendering.
type MetricGroup struct {
	Key       string
	Instances []MetricGroupInstance
}

// JobData holds all metric data of a HPC job.
//
// Metrics maps a metric name to its scope-organized data, for example
// jobData.Metrics["cpu_load"][MetricScopeNode]. Groups holds array-valued
// metric groups (filesystems, and later interconnects) whose members each
// carry their own scoped metrics; these cannot be expressed by the flat
// Metrics map because a plain map has nowhere to store per-instance identity.
//
// Custom (Un)MarshalJSON keep the on-disk/on-wire layout identical to the JSON
// schema: flat metrics are top-level scope objects and each group is a
// top-level array (see job-data.schema.json).
type JobData struct {
	Metrics map[string]ScopedMetrics
	Groups  []MetricGroup
}

// ScopedMetricStats maps a scope to its per-source statistical summaries.
type ScopedMetricStats map[MetricScope][]*ScopedStats

// ScopedStatsGroupInstance is the ScopedJobStats analogue of
// MetricGroupInstance: one named instance carrying scope-organized statistics.
type ScopedStatsGroupInstance struct {
	Name    string                       `json:"name"`
	Type    string                       `json:"type"`
	Metrics map[string]ScopedMetricStats `json:"-"`
}

// ScopedStatsGroup is an array-valued group of ScopedStatsGroupInstance.
type ScopedStatsGroup struct {
	Key       string
	Instances []ScopedStatsGroupInstance
}

// ScopedJobStats stores pre-computed statistics without the full time series
// data, reducing memory footprint when only aggregated values are needed. It
// mirrors JobData: flat metrics in Metrics, array-valued groups in Groups.
type ScopedJobStats struct {
	Metrics map[string]ScopedMetricStats
	Groups  []ScopedStatsGroup
}

// metricGroupKeys is the set of top-level JSON keys that are array-valued
// metric groups rather than scope objects. Adding a new group (e.g.
// "interconnects") is a single registration; no new types or codec are needed.
var metricGroupKeys = map[string]bool{
	"filesystems": true,
}

// RegisterMetricGroup registers an additional top-level key as an array-valued
// metric group so that it is (de)serialized into JobData.Groups.
func RegisterMetricGroup(key string) {
	metricGroupKeys[key] = true
}

// IsMetricGroupKey reports whether key denotes an array-valued metric group.
func IsMetricGroupKey(key string) bool {
	return metricGroupKeys[key]
}

// firstToken returns the first non-whitespace byte of a JSON value, or 0 if
// the value is empty. Used to distinguish an array group ('[') from a scope
// object ('{') when a key is not (yet) registered.
func firstToken(b []byte) byte {
	for _, c := range b {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return c
		}
	}
	return 0
}

// MarshalJSON renders JobData in the job-data schema layout: every flat metric
// as a top-level scope object, and every group as a top-level array of
// {name, type, <metric>: <scopeObject>, ...} instances. Per-metric values reuse
// the existing *JobMetric/Series marshalling (NaN -> null preserved).
func (jd JobData) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(jd.Metrics)+len(jd.Groups))

	for name, scopes := range jd.Metrics {
		b, err := json.Marshal(scopes)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}

	for _, group := range jd.Groups {
		arr := make([]json.RawMessage, 0, len(group.Instances))
		for _, inst := range group.Instances {
			obj := make(map[string]json.RawMessage, len(inst.Metrics)+2)
			name, err := json.Marshal(inst.Name)
			if err != nil {
				return nil, err
			}
			obj["name"] = name
			typ, err := json.Marshal(inst.Type)
			if err != nil {
				return nil, err
			}
			obj["type"] = typ
			for metric, scopes := range inst.Metrics {
				b, err := json.Marshal(scopes)
				if err != nil {
					return nil, err
				}
				obj[metric] = b
			}
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			arr = append(arr, b)
		}
		b, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		out[group.Key] = b
	}

	return json.Marshal(out)
}

// UnmarshalJSON parses the job-data schema layout into JobData. Top-level keys
// are dispatched by group-key registration; as a robustness fallback an
// unregistered key whose value is a JSON array is also treated as a group, so a
// reader can ingest a new array group before it is registered.
func (jd *JobData) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	jd.Metrics = make(map[string]ScopedMetrics, len(raw))
	jd.Groups = nil

	for key, msg := range raw {
		if IsMetricGroupKey(key) || firstToken(msg) == '[' {
			group, err := unmarshalMetricGroup(key, msg)
			if err != nil {
				return err
			}
			jd.Groups = append(jd.Groups, group)
			continue
		}

		var scopes ScopedMetrics
		if err := json.Unmarshal(msg, &scopes); err != nil {
			return err
		}
		jd.Metrics[key] = scopes
	}

	return nil
}

// unmarshalMetricGroup decodes a top-level array value into a MetricGroup,
// splitting each instance's name/type from its per-metric scope objects.
func unmarshalMetricGroup(key string, msg json.RawMessage) (MetricGroup, error) {
	group := MetricGroup{Key: key}

	var items []map[string]json.RawMessage
	if err := json.Unmarshal(msg, &items); err != nil {
		return group, err
	}

	for _, item := range items {
		inst := MetricGroupInstance{Metrics: make(map[string]ScopedMetrics, len(item))}
		for field, fmsg := range item {
			switch field {
			case "name":
				if err := json.Unmarshal(fmsg, &inst.Name); err != nil {
					return group, err
				}
			case "type":
				if err := json.Unmarshal(fmsg, &inst.Type); err != nil {
					return group, err
				}
			default:
				var scopes ScopedMetrics
				if err := json.Unmarshal(fmsg, &scopes); err != nil {
					return group, err
				}
				inst.Metrics[field] = scopes
			}
		}
		group.Instances = append(group.Instances, inst)
	}

	return group, nil
}

// FlatMap returns the flat scoped-metrics map, providing the pre-struct
// access shape (metric -> scope -> *JobMetric) for consumers that do not need
// the grouped metrics.
func (jd JobData) FlatMap() map[string]ScopedMetrics {
	return jd.Metrics
}

// AddGroupInstance appends a named instance (with its own scoped metrics) to the
// group identified by groupKey, creating the group if necessary. This is the
// converter seam used to assemble grouped data from flat, selector-style query
// results without the caller knowing the JobData layout.
func (jd *JobData) AddGroupInstance(groupKey, name, typ string, metrics map[string]ScopedMetrics) {
	inst := MetricGroupInstance{Name: name, Type: typ, Metrics: metrics}
	for i := range jd.Groups {
		if jd.Groups[i].Key == groupKey {
			jd.Groups[i].Instances = append(jd.Groups[i].Instances, inst)
			return
		}
	}
	jd.Groups = append(jd.Groups, MetricGroup{Key: groupKey, Instances: []MetricGroupInstance{inst}})
}

// MarshalJSON renders ScopedJobStats in the same top-level layout as JobData:
// flat metrics as scope objects, groups as top-level arrays of
// {name, type, <metric>: <scopeObject>} instances.
func (sjs ScopedJobStats) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(sjs.Metrics)+len(sjs.Groups))

	for name, scopes := range sjs.Metrics {
		b, err := json.Marshal(scopes)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}

	for _, group := range sjs.Groups {
		arr := make([]json.RawMessage, 0, len(group.Instances))
		for _, inst := range group.Instances {
			obj := make(map[string]json.RawMessage, len(inst.Metrics)+2)
			name, err := json.Marshal(inst.Name)
			if err != nil {
				return nil, err
			}
			obj["name"] = name
			typ, err := json.Marshal(inst.Type)
			if err != nil {
				return nil, err
			}
			obj["type"] = typ
			for metric, scopes := range inst.Metrics {
				b, err := json.Marshal(scopes)
				if err != nil {
					return nil, err
				}
				obj[metric] = b
			}
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			arr = append(arr, b)
		}
		b, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		out[group.Key] = b
	}

	return json.Marshal(out)
}

// UnmarshalJSON parses the top-level layout into ScopedJobStats, dispatching
// array-valued group keys into Groups (see JobData.UnmarshalJSON).
func (sjs *ScopedJobStats) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	sjs.Metrics = make(map[string]ScopedMetricStats, len(raw))
	sjs.Groups = nil

	for key, msg := range raw {
		if IsMetricGroupKey(key) || firstToken(msg) == '[' {
			group := ScopedStatsGroup{Key: key}
			var items []map[string]json.RawMessage
			if err := json.Unmarshal(msg, &items); err != nil {
				return err
			}
			for _, item := range items {
				inst := ScopedStatsGroupInstance{Metrics: make(map[string]ScopedMetricStats, len(item))}
				for field, fmsg := range item {
					switch field {
					case "name":
						if err := json.Unmarshal(fmsg, &inst.Name); err != nil {
							return err
						}
					case "type":
						if err := json.Unmarshal(fmsg, &inst.Type); err != nil {
							return err
						}
					default:
						var scopes ScopedMetricStats
						if err := json.Unmarshal(fmsg, &scopes); err != nil {
							return err
						}
						inst.Metrics[field] = scopes
					}
				}
				group.Instances = append(group.Instances, inst)
			}
			sjs.Groups = append(sjs.Groups, group)
			continue
		}

		var scopes ScopedMetricStats
		if err := json.Unmarshal(msg, &scopes); err != nil {
			return err
		}
		sjs.Metrics[key] = scopes
	}

	return nil
}

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
	ID         *string          `json:"id,omitempty"` // Optional ID (e.g., core ID, GPU ID)
	Hostname   string           `json:"hostname"`     // Source hostname
	Data       []Float          `json:"data"`         // Time series measurements
	Statistics MetricStatistics `json:"statistics"`   // Statistical summary (min/avg/max)
}

// ScopedStats contains statistical summaries for a specific scope (e.g., one node, one socket).
// Used when full time series data isn't needed, only the aggregated statistics.
type ScopedStats struct {
	Hostname string            `json:"hostname"`     // Source hostname
	ID       *string           `json:"id,omitempty"` // Optional scope ID
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
	sizeScopes := func(scopes ScopedMetrics) {
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

	for _, scopes := range jd.Metrics {
		sizeScopes(scopes)
	}
	for _, group := range jd.Groups {
		for _, inst := range group.Instances {
			for _, scopes := range inst.Metrics {
				sizeScopes(scopes)
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
	scopes, ok := jd.Metrics[metric]
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

		n, m := 0, len(series[0].Data)
		for _, s := range series {
			if len(s.Data) > n {
				n = len(s.Data)
			}
			if len(s.Data) < m {
				m = len(s.Data)
			}
		}

		i, data := 0, make([]Float, n)
		for ; i < m; i++ {
			x := Float(0.0)
			for _, s := range series {
				x += s.Data[i]
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
	roundScopes := func(scopes ScopedMetrics) {
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

	for _, scopes := range jd.Metrics {
		roundScopes(scopes)
	}
	for _, group := range jd.Groups {
		for _, inst := range group.Instances {
			for _, scopes := range inst.Metrics {
				roundScopes(scopes)
			}
		}
	}
}

func (sjs *ScopedJobStats) RoundScopedMetricStats() {
	// TODO: Make Digit-Precision Configurable? (Currently: Fixed to 2 Digits)
	roundScopes := func(scopes ScopedMetricStats) {
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

	for _, scopes := range sjs.Metrics {
		roundScopes(scopes)
	}
	for _, group := range sjs.Groups {
		for _, inst := range group.Instances {
			for _, scopes := range inst.Metrics {
				roundScopes(scopes)
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
