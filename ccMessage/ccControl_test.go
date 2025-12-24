// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

func TestNewGetControl(t *testing.T) {
	msg, err := NewGetControl("test_control", map[string]string{"host": "node001"}, map[string]string{"unit": "none"}, time.Now())
	if err != nil {
		t.Fatalf("NewGetControl failed: %v", err)
	}

	if msg.Name() != "test_control" {
		t.Errorf("Expected name 'test_control', got '%s'", msg.Name())
	}

	if method, ok := msg.GetTag("method"); !ok || method != "GET" {
		t.Errorf("Expected method tag 'GET', got '%s' (ok=%v)", method, ok)
	}

	if _, ok := msg.GetField("control"); !ok {
		t.Error("Expected 'control' field to exist")
	}

	if !msg.IsControl() {
		t.Error("Expected IsControl() to return true")
	}
}

func TestNewPutControl(t *testing.T) {
	msg, err := NewPutControl("test_control", map[string]string{"host": "node001"}, map[string]string{"unit": "none"}, "test_value", time.Now())
	if err != nil {
		t.Fatalf("NewPutControl failed: %v", err)
	}

	if msg.Name() != "test_control" {
		t.Errorf("Expected name 'test_control', got '%s'", msg.Name())
	}

	if method, ok := msg.GetTag("method"); !ok || method != "PUT" {
		t.Errorf("Expected method tag 'PUT', got '%s' (ok=%v)", method, ok)
	}

	if value, ok := msg.GetField("control"); !ok || value != "test_value" {
		t.Errorf("Expected control field 'test_value', got '%v' (ok=%v)", value, ok)
	}

	if !msg.IsControl() {
		t.Error("Expected IsControl() to return true")
	}

	if value, ok := msg.GetControlValue(); !ok || value != "test_value" {
		t.Errorf("Expected GetControlValue() to return 'test_value', got '%s' (ok=%v)", value, ok)
	}

	if method, ok := msg.GetControlMethod(); !ok || method != "PUT" {
		t.Errorf("Expected GetControlMethod() to return 'PUT', got '%s' (ok=%v)", method, ok)
	}
}

func TestIsControl_WithoutMethodTag(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]interface{}{"control": "value"}, time.Now())

	if msg.IsControl() {
		t.Error("Expected IsControl() to return false when method tag is missing")
	}
}

func TestIsControl_WithInvalidMethod(t *testing.T) {
	msg, _ := NewMessage("test", map[string]string{"method": "DELETE"}, nil, map[string]interface{}{"control": "value"}, time.Now())

	if msg.IsControl() {
		t.Error("Expected IsControl() to return false for invalid method")
	}
}

func TestIsControl_WithNonStringValue(t *testing.T) {
	msg, _ := NewMessage("test", map[string]string{"method": "PUT"}, nil, map[string]interface{}{"control": 123}, time.Now())

	if msg.IsControl() {
		t.Error("Expected IsControl() to return false for non-string control value")
	}
}

func TestGetControlValue_EmptyString(t *testing.T) {
	msg, _ := NewGetControl("test", nil, nil, time.Now())

	if value, ok := msg.GetControlValue(); !ok || value != "" {
		t.Errorf("Expected empty string for GET control, got '%s' (ok=%v)", value, ok)
	}
}

func TestGetControlMethod_GET(t *testing.T) {
	msg, _ := NewGetControl("test", nil, nil, time.Now())

	if method, ok := msg.GetControlMethod(); !ok || method != "GET" {
		t.Errorf("Expected 'GET', got '%s' (ok=%v)", method, ok)
	}
}
