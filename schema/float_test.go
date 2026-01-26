// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema_test

import (
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestFloat(t *testing.T) {
	// Test IsNaN
	f := schema.NaN
	if !f.IsNaN() {
		t.Error("expected NaN.IsNaN() to return true")
	}

	f = schema.Float(3.14)
	if f.IsNaN() {
		t.Error("expected Float(3.14).IsNaN() to return false")
	}

	// Test Double
	if f.Double() != 3.14 {
		t.Errorf("expected Double() to return 3.14, got %f", f.Double())
	}

	// Test MarshalJSON for NaN
	nanJSON, err := schema.NaN.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(nanJSON) != "null" {
		t.Errorf("expected 'null' for NaN, got %s", nanJSON)
	}

	// Test MarshalJSON for normal value
	normalJSON, err := schema.Float(3.142).MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(normalJSON) != "3.14" {
		t.Errorf("expected '3.14', got %s", normalJSON)
	}

	// Test UnmarshalJSON for null
	var f2 schema.Float
	if err := f2.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !f2.IsNaN() {
		t.Error("expected unmarshaled null to be NaN")
	}

	// Test UnmarshalJSON for number
	if err := f2.UnmarshalJSON([]byte("5.678")); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if f2.Double() != 5.678 {
		t.Errorf("expected 5.678, got %f", f2.Double())
	}

	// Test ConvertToFloat
	converted := schema.ConvertToFloat(-1.0)
	if !converted.IsNaN() {
		t.Error("expected ConvertToFloat(-1.0) to return NaN")
	}
	converted = schema.ConvertToFloat(10.5)
	if converted.Double() != 10.5 {
		t.Errorf("expected ConvertToFloat(10.5) to return 10.5, got %f", converted.Double())
	}

	// Test FloatArray MarshalJSON
	arr := schema.FloatArray{schema.Float(1.0), schema.NaN, schema.Float(3.0)}
	arrJSON, err := arr.MarshalJSON()
	if err != nil {
		t.Fatalf("FloatArray MarshalJSON failed: %v", err)
	}
	if string(arrJSON) != "[1.000,null,3.000]" {
		t.Errorf("expected '[1.000,null,3.000]', got %s", arrJSON)
	}
}
