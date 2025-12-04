// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

func TestNewMetric_Float(t *testing.T) {
	msg, err := NewMetric("cpu_usage", map[string]string{"type": "node"}, map[string]string{"unit": "percent"}, 75.5, time.Now())
	if err != nil {
		t.Fatalf("NewMetric failed: %v", err)
	}

	if msg.Name() != "cpu_usage" {
		t.Errorf("Expected name 'cpu_usage', got '%s'", msg.Name())
	}

	if value, ok := msg.GetField("value"); !ok || value != 75.5 {
		t.Errorf("Expected value field 75.5, got '%v' (ok=%v)", value, ok)
	}

	if !msg.IsMetric() {
		t.Error("Expected IsMetric() to return true")
	}

	if msg.GetMetricValue() != 75.5 {
		t.Errorf("Expected GetMetricValue() to return 75.5, got '%v'", msg.GetMetricValue())
	}
}

func TestNewMetric_Int(t *testing.T) {
	msg, err := NewMetric("mem_used", map[string]string{"type": "node"}, map[string]string{"unit": "bytes"}, int64(1024), time.Now())
	if err != nil {
		t.Fatalf("NewMetric with int failed: %v", err)
	}

	if !msg.IsMetric() {
		t.Error("Expected IsMetric() to return true for int value")
	}

	if msg.GetMetricValue() != int64(1024) {
		t.Errorf("Expected GetMetricValue() to return 1024, got '%v'", msg.GetMetricValue())
	}
}

func TestNewMetric_Uint(t *testing.T) {
	msg, err := NewMetric("packet_count", map[string]string{"type": "node"}, map[string]string{"unit": "count"}, uint64(999999), time.Now())
	if err != nil {
		t.Fatalf("NewMetric with uint failed: %v", err)
	}

	if !msg.IsMetric() {
		t.Error("Expected IsMetric() to return true for uint value")
	}

	if msg.GetMetricValue() != uint64(999999) {
		t.Errorf("Expected GetMetricValue() to return 999999, got '%v'", msg.GetMetricValue())
	}
}

func TestIsMetric_WithStringValue(t *testing.T) {
	// Metrics should not have string values according to IsMetric logic
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"value": "string_value"}, time.Now())

	if msg.IsMetric() {
		t.Error("Expected IsMetric() to return false for string value")
	}
}

func TestIsMetric_WithoutValueField(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"event": "test"}, time.Now())

	if msg.IsMetric() {
		t.Error("Expected IsMetric() to return false when value field is missing")
	}
}

func TestGetMetricValue_NonMetric(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"event": "test"}, time.Now())

	if value := msg.GetMetricValue(); value != nil {
		t.Errorf("Expected nil for non-metric, got '%v'", value)
	}
}

func TestNewMetric_Zero(t *testing.T) {
	msg, err := NewMetric("zero_metric", nil, nil, 0.0, time.Now())
	if err != nil {
		t.Fatalf("NewMetric with zero failed: %v", err)
	}

	if !msg.IsMetric() {
		t.Error("Expected IsMetric() to return true for zero value")
	}

	if msg.GetMetricValue() != 0.0 {
		t.Errorf("Expected GetMetricValue() to return 0.0, got '%v'", msg.GetMetricValue())
	}
}

func TestNewMetric_NegativeValue(t *testing.T) {
	msg, err := NewMetric("temperature", nil, map[string]string{"unit": "celsius"}, -15.5, time.Now())
	if err != nil {
		t.Fatalf("NewMetric with negative value failed: %v", err)
	}

	if msg.GetMetricValue() != -15.5 {
		t.Errorf("Expected GetMetricValue() to return -15.5, got '%v'", msg.GetMetricValue())
	}
}
