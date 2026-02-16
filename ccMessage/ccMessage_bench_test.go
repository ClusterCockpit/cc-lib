// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

var (
	benchmarkResult       any
	benchmarkStringResult string
	benchmarkBoolResult   bool
	benchmarkBytesResult  []byte
)

func BenchmarkConvertField_Float64(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(float64(123.456))
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Int64(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(int64(123456))
	}
	benchmarkResult = r
}

func BenchmarkConvertField_String(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField("test string value")
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Int(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(int(123456))
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Float32(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(float32(123.456))
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Uint64(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(uint64(123456))
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Bool(b *testing.B) {
	var r any
	for i := 0; i < b.N; i++ {
		r = convertField(true)
	}
	benchmarkResult = r
}

func BenchmarkConvertField_Bytes(b *testing.B) {
	var r any
	data := []byte("test string value")
	for i := 0; i < b.N; i++ {
		r = convertField(data)
	}
	benchmarkResult = r
}

func BenchmarkConvertField_PointerInt64(b *testing.B) {
	var r any
	val := int64(123456)
	for i := 0; i < b.N; i++ {
		r = convertField(&val)
	}
	benchmarkResult = r
}

func BenchmarkNewMessage_Simple(b *testing.B) {
	ts := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = NewMessage(
			"test_metric",
			map[string]string{"type": "node"},
			map[string]string{"unit": "bytes"},
			map[string]any{"value": 123.456},
			ts,
		)
	}
}

func BenchmarkNewMessage_WithMultipleTags(b *testing.B) {
	ts := time.Now()
	tags := map[string]string{
		"type":     "node",
		"hostname": "node001",
		"cluster":  "test",
		"rack":     "rack1",
	}
	for i := 0; i < b.N; i++ {
		_, _ = NewMessage(
			"test_metric",
			tags,
			map[string]string{"unit": "bytes"},
			map[string]any{"value": 123.456},
			ts,
		)
	}
}

func BenchmarkNewMetric(b *testing.B) {
	ts := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = NewMetric(
			"cpu_usage",
			map[string]string{"type": "node"},
			map[string]string{"unit": "percent"},
			75.5,
			ts,
		)
	}
}

func BenchmarkNewEvent(b *testing.B) {
	ts := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = NewEvent(
			"node_down",
			map[string]string{"severity": "critical"},
			nil,
			"Node node001 is unreachable",
			ts,
		)
	}
}

func BenchmarkNewLog(b *testing.B) {
	ts := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = NewLog(
			"app_log",
			map[string]string{"level": "error"},
			map[string]string{"source": "backend"},
			"Database connection failed",
			ts,
		)
	}
}

func BenchmarkFromMessage(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromMessage(msg)
	}
}

func BenchmarkToLineProtocol_NoMeta(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	var r string
	for i := 0; i < b.N; i++ {
		r = msg.ToLineProtocol(map[string]bool{})
	}
	benchmarkStringResult = r
}

func BenchmarkToLineProtocol_WithMeta(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	var r string
	for i := 0; i < b.N; i++ {
		r = msg.ToLineProtocol(map[string]bool{"unit": true})
	}
	benchmarkStringResult = r
}

func BenchmarkBytes(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	var r []byte
	for i := 0; i < b.N; i++ {
		r, _ = msg.(*ccMessage).Bytes()
	}
	benchmarkBytesResult = r
}

func BenchmarkToJSON_NoMeta(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.ToJSON(map[string]bool{})
	}
}

func BenchmarkToJSON_WithMeta(b *testing.B) {
	msg, _ := NewMetric(
		"cpu_usage",
		map[string]string{"type": "node", "hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Now(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.ToJSON(map[string]bool{"unit": true})
	}
}

func BenchmarkIsMetric(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	var r bool
	for i := 0; i < b.N; i++ {
		r = msg.IsMetric()
	}
	benchmarkBoolResult = r
}

func BenchmarkIsEvent(b *testing.B) {
	msg, _ := NewEvent("test", nil, nil, "event data", time.Now())

	b.ResetTimer()
	var r bool
	for i := 0; i < b.N; i++ {
		r = msg.IsEvent()
	}
	benchmarkBoolResult = r
}

func BenchmarkGetMetricValue(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.GetMetricValue()
	}
}

func BenchmarkMessageType(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.MessageType()
	}
}

func BenchmarkAddTag(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.AddTag("key", "value")
	}
}

func BenchmarkGetTag(b *testing.B) {
	msg, _ := NewMetric("test", map[string]string{"key": "value"}, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.GetTag("key")
	}
}

func BenchmarkAddField(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.AddField("extra", 456.789)
	}
}

func BenchmarkGetField(b *testing.B) {
	msg, _ := NewMetric("test", nil, nil, 123.456, time.Now())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.GetField("value")
	}
}

func BenchmarkFromBytes_Single(b *testing.B) {
	data := []byte("cpu_usage,type=node,hostname=node001 value=75.5 1234567890000000000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromBytes(data)
	}
}

func BenchmarkFromBytes_Multiple(b *testing.B) {
	data := []byte(`cpu_usage,type=node value=75.5 1234567890000000000
mem_used,type=node value=1024 1234567890000000000
net_bytes,type=node value=9999 1234567890000000000`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FromBytes(data)
	}
}
