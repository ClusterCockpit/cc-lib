// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"time"
)

// NewMetric creates a new metric message.
// Metrics represent numerical measurements from the monitored system, such as CPU usage,
// memory consumption, network throughput, etc. This is the primary message type in ClusterCockpit
// for performance monitoring data.
//
// Parameters:
//   - name: The metric name (e.g., "cpu_usage", "mem_used", "flops_any")
//   - tags: Optional tags for categorizing the metric (e.g., "type": "node", "hostname": "node001")
//   - meta: Optional metadata information (e.g., "unit": "bytes", "scope": "node")
//   - value: The metric value (can be int, float, uint, or other numeric types)
//   - tm: Timestamp when the metric was collected
//
// Returns a CCMessage with the "value" field set to the provided metric value.
// Note: Unlike events and logs, metric values should be numeric, not strings.
func NewMetric(name string,
	tags map[string]string,
	meta map[string]string,
	value any,
	tm time.Time,
) (CCMessage, error) {
	return NewMessage(name, tags, meta, map[string]any{"value": value}, tm)
}

func (m *ccMessage) IsMetric() bool {
	if v, ok := m.GetField("value"); ok {
		if _, ok := v.(string); !ok {
			return true
		}
	}
	return false
}

func (m *ccMessage) GetMetricValue() (any, bool) {
	if m.IsMetric() {
		if v, ok := m.GetField("value"); ok {
			return v, true
		}
	}
	return nil, false
}
