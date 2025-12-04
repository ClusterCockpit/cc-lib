// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

func TestNewLog(t *testing.T) {
	logPayload := "Application started successfully"
	msg, err := NewLog("app_log", map[string]string{"level": "info"}, map[string]string{"source": "app"}, logPayload, time.Now())
	if err != nil {
		t.Fatalf("NewLog failed: %v", err)
	}

	if msg.Name() != "app_log" {
		t.Errorf("Expected name 'app_log', got '%s'", msg.Name())
	}

	if value, ok := msg.GetField("log"); !ok || value != logPayload {
		t.Errorf("Expected log field '%s', got '%v' (ok=%v)", logPayload, value, ok)
	}

	if !msg.IsLog() {
		t.Error("Expected IsLog() to return true")
	}

	if msg.GetLogValue() != logPayload {
		t.Errorf("Expected GetLogValue() to return '%s', got '%s'", logPayload, msg.GetLogValue())
	}
}

func TestNewLog_EmptyMessage(t *testing.T) {
	msg, err := NewLog("empty_log", nil, nil, "", time.Now())
	if err != nil {
		t.Fatalf("NewLog with empty message failed: %v", err)
	}

	if !msg.IsLog() {
		t.Error("Expected IsLog() to return true even with empty message")
	}

	if msg.GetLogValue() != "" {
		t.Errorf("Expected empty string, got '%s'", msg.GetLogValue())
	}
}

func TestIsLog_WithNonStringValue(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"log": 123}, time.Now())

	if msg.IsLog() {
		t.Error("Expected IsLog() to return false for non-string log value")
	}
}

func TestIsLog_WithoutLogField(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"value": 1.0}, time.Now())

	if msg.IsLog() {
		t.Error("Expected IsLog() to return false when log field is missing")
	}
}

func TestGetLogValue_NonLog(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"value": 1.0}, time.Now())

	if value := msg.GetLogValue(); value != "" {
		t.Errorf("Expected empty string for non-log, got '%s'", value)
	}
}

func TestNewLog_MultilineMessage(t *testing.T) {
	multilineLog := "Line 1\nLine 2\nLine 3"
	msg, err := NewLog("multiline_log", nil, nil, multilineLog, time.Now())
	if err != nil {
		t.Fatalf("NewLog with multiline message failed: %v", err)
	}

	if msg.GetLogValue() != multilineLog {
		t.Errorf("Expected multiline log to be preserved, got '%s'", msg.GetLogValue())
	}
}
