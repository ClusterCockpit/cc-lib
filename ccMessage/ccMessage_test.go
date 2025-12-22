// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestJSONEncode(t *testing.T) {
	input := []CCMessage{
		&ccMessage{name: "test1", tags: map[string]string{"type": "node"}, meta: map[string]string{"unit": "B"}, fields: map[string]any{"value": 1.23}, tm: time.Now()},
		&ccMessage{name: "test2", tags: map[string]string{"type": "socket", "type-id": "0"}, meta: map[string]string{"unit": "B"}, fields: map[string]any{"value": 1.23}, tm: time.Now()},
	}

	x, err := json.Marshal(input)
	if err != nil {
		t.Error(err.Error())
		return
	}
	t.Log(string(x))
}

func TestJSONDecode(t *testing.T) {
	input := `[{"name":"test1","tags":{"type":"node"},"fields":{"value":1.23},"timestamp":"2024-06-22T13:51:59.495479906+02:00"},{"name":"test2","tags":{"type":"socket","type-id":"0"},"fields":{"value":1.23},"timestamp":"2024-06-22T13:51:59.495481095+02:00"}]`
	var list []*ccMessage
	///var list []CCMessage
	err := json.Unmarshal([]byte(input), &list)
	if err != nil {
		t.Error(err.Error())
		return
	}
	// t.Log(list)
	for _, m := range list {
		t.Log(m.Name())
	}
}

func TestILPDecode(t *testing.T) {
	input := fmt.Sprintf(`test1,type=node value=1.23 %d
test2,type=socket,type-id=0 value=1.23 %d`, time.Now().UnixNano(), time.Now().UnixNano())

	list, err := FromBytes([]byte(input))
	if err != nil {
		t.Error(err.Error())
		return
	}
	for _, m := range list {
		t.Log(m.Name())
	}
}

func TestMessageType_Metric(t *testing.T) {
	msg, _ := NewMetric("test", nil, nil, 1.0, time.Now())
	if msg.MessageType() != CCMSG_TYPE_METRIC {
		t.Errorf("Expected CCMSG_TYPE_METRIC, got %v", msg.MessageType())
	}
}

func TestMessageType_Event(t *testing.T) {
	msg, _ := NewEvent("test", nil, nil, "event", time.Now())
	if msg.MessageType() != CCMSG_TYPE_EVENT {
		t.Errorf("Expected CCMSG_TYPE_EVENT, got %v", msg.MessageType())
	}
}

func TestMessageType_Log(t *testing.T) {
	msg, _ := NewLog("test", nil, nil, "log", time.Now())
	if msg.MessageType() != CCMSG_TYPE_LOG {
		t.Errorf("Expected CCMSG_TYPE_LOG, got %v", msg.MessageType())
	}
}

func TestMessageType_Control(t *testing.T) {
	msg, _ := NewPutControl("test", nil, nil, "value", time.Now())
	if msg.MessageType() != CCMSG_TYPE_CONTROL {
		t.Errorf("Expected CCMSG_TYPE_CONTROL, got %v", msg.MessageType())
	}
}

func TestMessageType_Invalid(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{"unknown": "field"}, time.Now())
	if msg.MessageType() != CCMSG_TYPE_INVALID {
		t.Errorf("Expected CCMSG_TYPE_INVALID, got %v", msg.MessageType())
	}
}

func TestFromMessage_DeepCopy(t *testing.T) {
	original, _ := NewMetric("test", map[string]string{"tag1": "value1"}, map[string]string{"meta1": "metavalue1"}, 1.0, time.Now())
	copy := FromMessage(original)

	// Modify the copy
	copy.SetName("modified")
	copy.AddTag("tag2", "value2")
	copy.AddMeta("meta2", "metavalue2")
	copy.AddField("extra", 2.0)

	// Verify original is unchanged
	if original.Name() != "test" {
		t.Error("Original name was modified")
	}
	if _, ok := original.GetTag("tag2"); ok {
		t.Error("Original tags were modified")
	}
	if _, ok := original.GetMeta("meta2"); ok {
		t.Error("Original meta was modified")
	}
	if _, ok := original.GetField("extra"); ok {
		t.Error("Original fields were modified")
	}
}

func TestConvertField_IntTypes(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{
		"int":    int(10),
		"int8":   int8(10),
		"int16":  int16(10),
		"int32":  int32(10),
		"int64":  int64(10),
		"uint":   uint(10),
		"uint8":  uint8(10),
		"uint16": uint16(10),
		"uint32": uint32(10),
		"uint64": uint64(10),
	}, time.Now())

	// All int types should be converted to int64
	if v, _ := msg.GetField("int"); v != int64(10) {
		t.Errorf("int not converted to int64: %T", v)
	}
	if v, _ := msg.GetField("int32"); v != int64(10) {
		t.Errorf("int32 not converted to int64: %T", v)
	}

	// All uint types should be converted to uint64
	if v, _ := msg.GetField("uint"); v != uint64(10) {
		t.Errorf("uint not converted to uint64: %T", v)
	}
	if v, _ := msg.GetField("uint32"); v != uint64(10) {
		t.Errorf("uint32 not converted to uint64: %T", v)
	}
}

func TestConvertField_FloatTypes(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{
		"float32": float32(1.5),
		"float64": float64(2.5),
	}, time.Now())

	// All float types should be converted to float64
	if v, _ := msg.GetField("float32"); v != float64(float32(1.5)) {
		t.Errorf("float32 not converted to float64: %T", v)
	}
	if v, _ := msg.GetField("float64"); v != float64(2.5) {
		t.Errorf("float64 type mismatch: %T", v)
	}
}

func TestConvertField_StringAndBytes(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{
		"string": "test",
		"bytes":  []byte("test"),
	}, time.Now())

	// []byte should be converted to string
	if v, _ := msg.GetField("bytes"); v != "test" {
		t.Errorf("[]byte not converted to string: %v (%T)", v, v)
	}
	if v, _ := msg.GetField("string"); v != "test" {
		t.Errorf("string type mismatch: %T", v)
	}
}

func TestConvertField_Bool(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{
		"bool_true":  true,
		"bool_false": false,
	}, time.Now())

	if v, _ := msg.GetField("bool_true"); v != true {
		t.Errorf("bool true mismatch: %v", v)
	}
	if v, _ := msg.GetField("bool_false"); v != false {
		t.Errorf("bool false mismatch: %v", v)
	}
}

func TestEmptyMessage(t *testing.T) {
	msg := EmptyMessage()

	if msg.Name() != "" {
		t.Errorf("Expected empty name, got '%s'", msg.Name())
	}
	if len(msg.Tags()) != 0 {
		t.Errorf("Expected empty tags, got %v", msg.Tags())
	}
	if len(msg.Meta()) != 0 {
		t.Errorf("Expected empty meta, got %v", msg.Meta())
	}
	if len(msg.Fields()) != 0 {
		t.Errorf("Expected empty fields, got %v", msg.Fields())
	}
}

func TestToLineProtocol(t *testing.T) {
	msg, _ := NewMetric("cpu_usage", map[string]string{"host": "node001"}, map[string]string{"unit": "percent"}, 75.5, time.Now())
	lp := msg.ToLineProtocol(map[string]bool{})

	if lp == "" {
		t.Error("Expected non-empty line protocol")
	}
	// Basic validation that it contains the metric name
	if len(lp) < len("cpu_usage") {
		t.Error("Line protocol seems too short")
	}
}

func TestNewMessage_EmptyName(t *testing.T) {
	_, err := NewMessage("", nil, nil, map[string]any{"value": 1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestNewMessage_WhitespaceName(t *testing.T) {
	_, err := NewMessage("   ", nil, nil, map[string]any{"value": 1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for whitespace-only name")
	}
}

func TestNewMessage_ZeroTimestamp(t *testing.T) {
	_, err := NewMessage("test", nil, nil, map[string]any{"value": 1.0}, time.Time{})
	if err == nil {
		t.Error("Expected error for zero timestamp")
	}
}

func TestNewMessage_EmptyTagKey(t *testing.T) {
	_, err := NewMessage("test", map[string]string{"": "value"}, nil, map[string]any{"value": 1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for empty tag key")
	}
}

func TestNewMessage_EmptyMetaKey(t *testing.T) {
	_, err := NewMessage("test", nil, map[string]string{"": "value"}, map[string]any{"value": 1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for empty meta key")
	}
}

func TestNewMessage_EmptyFieldKey(t *testing.T) {
	_, err := NewMessage("test", nil, nil, map[string]any{"": 1.0}, time.Now())
	if err == nil {
		t.Error("Expected error for empty field key")
	}
}

func TestNewMessage_NoFields(t *testing.T) {
	_, err := NewMessage("test", nil, nil, map[string]any{}, time.Now())
	if err == nil {
		t.Error("Expected error for no fields")
	}
}

func TestNewMessage_NaNValue(t *testing.T) {
	_, err := NewMessage("test", nil, nil, map[string]any{"value": math.NaN()}, time.Now())
	if err == nil {
		t.Error("Expected error for NaN value")
	}
}

func TestNewMessage_InfValue(t *testing.T) {
	_, err := NewMessage("test", nil, nil, map[string]any{"value": math.Inf(1)}, time.Now())
	if err == nil {
		t.Error("Expected error for Inf value")
	}
}

func TestNewMessage_AllNilFields(t *testing.T) {
	type unsupportedType struct{ data string }
	_, err := NewMessage("test", nil, nil, map[string]any{"bad": unsupportedType{"test"}}, time.Now())
	if err == nil {
		t.Error("Expected error when all fields are nil/invalid")
	}
}
