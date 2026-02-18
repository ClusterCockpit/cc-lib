// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"
)

func TestNewQuery(t *testing.T) {
	queryString := "SELECT * FROM metrics WHERE host='node001'"
	msg, err := NewQuery("db_query", map[string]string{"database": "metrics"}, map[string]string{"engine": "postgres"}, queryString, time.Now())
	if err != nil {
		t.Fatalf("NewQuery failed: %v", err)
	}

	if msg.Name() != "db_query" {
		t.Errorf("Expected name 'db_query', got '%s'", msg.Name())
	}

	if value, ok := msg.GetField("query"); !ok || value != queryString {
		t.Errorf("Expected query field '%s', got '%v' (ok=%v)", queryString, value, ok)
	}

	if !msg.IsQuery() {
		t.Error("Expected IsQuery() to return true")
	}

	if value, ok := msg.GetQueryValue(); !ok || value != queryString {
		t.Errorf("Expected GetQueryValue() to return '%s', got '%s' (ok=%v)", queryString, value, ok)
	}
}

func TestNewQuery_EmptyQuery(t *testing.T) {
	msg, err := NewQuery("empty_query", nil, nil, "", time.Now())
	if err != nil {
		t.Fatalf("NewQuery with empty string failed: %v", err)
	}

	if !msg.IsQuery() {
		t.Error("Expected IsQuery() to return true even with empty query")
	}

	if value, ok := msg.GetQueryValue(); !ok || value != "" {
		t.Errorf("Expected empty string, got '%s' (ok=%v)", value, ok)
	}
}

func TestIsQuery_WithNonStringValue(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{"query": 123}, time.Now())

	if msg.IsQuery() {
		t.Error("Expected IsQuery() to return false for non-string query value")
	}
}

func TestIsQuery_WithoutQueryField(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{"value": 1.0}, time.Now())

	if msg.IsQuery() {
		t.Error("Expected IsQuery() to return false when query field is missing")
	}
}

func TestGetQueryValue_NonQuery(t *testing.T) {
	msg, _ := NewMessage("test", nil, nil, map[string]any{"value": 1.0}, time.Now())

	if value, ok := msg.GetQueryValue(); ok {
		t.Errorf("Expected ok=false for non-query, got value='%s' (ok=%v)", value, ok)
	}
}

func TestNewQuery_ComplexQuery(t *testing.T) {
	complexQuery := `
		SELECT m.name, AVG(m.value) as avg_value
		FROM metrics m
		WHERE m.timestamp > NOW() - INTERVAL '1 hour'
		GROUP BY m.name
		ORDER BY avg_value DESC
		LIMIT 10
	`
	msg, err := NewQuery("analytics", nil, nil, complexQuery, time.Now())
	if err != nil {
		t.Fatalf("NewQuery with complex query failed: %v", err)
	}

	if value, ok := msg.GetQueryValue(); !ok || value != complexQuery {
		t.Errorf("Expected complex query to be preserved, got ok=%v", ok)
	}
}
