// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding/json"
	"testing"
)

func TestSeriesJSON_PreservesFloat64Data(t *testing.T) {
	for _, value := range []Float{16777217, -16777217, 1e40, -1e40} {
		series := Series{Hostname: "node001", Data: []Float{value}}
		encoded, err := json.Marshal(&series)
		if err != nil {
			t.Errorf("finite value %v rejected: %v", value, err)
			continue
		}
		var restored Series
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		if len(restored.Data) != 1 || restored.Data[0] != value {
			t.Errorf("float64 changed: %v -> %v", value, restored.Data)
		}
	}
}

func TestSeriesJSON_QuotesStringFields(t *testing.T) {
	for _, value := range []string{"gpu\\0", "quote\"id", "line\nnext", "control\x00\a\v", string([]byte{0xff})} {
		series := Series{Hostname: value, ID: &value, Data: []Float{1}}
		encoded, err := json.Marshal(&series)
		if err != nil {
			t.Errorf("string serialization failed: %v", err)
			continue
		}
		var got Series
		if err := json.Unmarshal(encoded, &got); err != nil {
			t.Fatal(err)
		}
		// encoding/json defines replacement of invalid UTF-8; compare its string oracle.
		reference, _ := json.Marshal(value)
		var want string
		if err := json.Unmarshal(reference, &want); err != nil {
			t.Fatal(err)
		}
		if got.Hostname != want || got.ID == nil || *got.ID != want {
			t.Errorf("string identity lost: host=%q ID=%v want=%q", got.Hostname, got.ID, want)
		}
	}
}
