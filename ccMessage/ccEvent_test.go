// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	eventPayload := "System maintenance scheduled"
	msg, err := NewEvent("maintenance", map[string]string{"severity": "info"}, map[string]string{"source": "admin"}, eventPayload, time.Now())
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if msg.Name() != "maintenance" {
		t.Errorf("Expected name 'maintenance', got '%s'", msg.Name())
	}

	if value, ok := msg.GetField("event"); !ok || value != eventPayload {
		t.Errorf("Expected event field '%s', got '%v' (ok=%v)", eventPayload, value, ok)
	}

	if !msg.IsEvent() {
		t.Error("Expected IsEvent() to return true")
	}

	if msg.GetEventValue() != eventPayload {
		t.Errorf("Expected GetEventValue() to return '%s', got '%s'", eventPayload, msg.GetEventValue())
	}
}

func TestNewEvent_EmptyPayload(t *testing.T) {
	msg, err := NewEvent("test_event", nil, nil, "", time.Now())
	if err != nil {
		t.Fatalf("NewEvent with empty payload failed: %v", err)
	}

	if !msg.IsEvent() {
		t.Error("Expected IsEvent() to return true even with empty payload")
	}

	if msg.GetEventValue() != "" {
		t.Errorf("Expected empty string, got '%s'", msg.GetEventValue())
	}
}

func TestIsEvent_WithNonStringValue(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"event": 123}, time.Now())

	if msg.IsEvent() {
		t.Error("Expected IsEvent() to return false for non-string event value")
	}
}

func TestIsEvent_WithoutEventField(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"value": 1.0}, time.Now())

	if msg.IsEvent() {
		t.Error("Expected IsEvent() to return false when event field is missing")
	}
}

func TestGetEventValue_NonEvent(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"value": 1.0}, time.Now())

	if value := msg.GetEventValue(); value != "" {
		t.Errorf("Expected empty string for non-event, got '%s'", value)
	}
}

func TestNewEvent_WithJSONPayload(t *testing.T) {
	jsonPayload := `{"status": "ok", "message": "test"}`
	msg, err := NewEvent("api_response", nil, nil, jsonPayload, time.Now())
	if err != nil {
		t.Fatalf("NewEvent with JSON payload failed: %v", err)
	}

	if msg.GetEventValue() != jsonPayload {
		t.Errorf("Expected JSON payload to be preserved, got '%s'", msg.GetEventValue())
	}
}
