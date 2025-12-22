// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"time"
)

// NewGetControl creates a new control message with a GET method.
// Control messages are used to request or set configuration values in the ClusterCockpit system.
// A GET control message requests the current value of the specified control parameter.
//
// Parameters:
//   - name: The name of the control parameter to query
//   - tags: Optional tags for categorizing the control message
//   - meta: Optional metadata information
//   - tm: Timestamp when the control message was created
//
// Returns a CCMessage with the "control" field set to an empty string and a "method" tag set to "GET".
func NewGetControl(name string,
	tags map[string]string,
	meta map[string]string,
	tm time.Time,
) (CCMessage, error) {
	m, err := NewMessage(name, tags, meta, map[string]any{"control": ""}, tm)
	if err == nil {
		m.AddTag("method", "GET")
	}
	return m, err
}

// NewPutControl creates a new control message with a PUT method.
// Control messages are used to request or set configuration values in the ClusterCockpit system.
// A PUT control message sets a new value for the specified control parameter.
//
// Parameters:
//   - name: The name of the control parameter to set
//   - tags: Optional tags for categorizing the control message
//   - meta: Optional metadata information
//   - value: The new value to set for the control parameter
//   - tm: Timestamp when the control message was created
//
// Returns a CCMessage with the "control" field set to the provided value and a "method" tag set to "PUT".
func NewPutControl(name string,
	tags map[string]string,
	meta map[string]string,
	value string,
	tm time.Time,
) (CCMessage, error) {
	m, err := NewMessage(name, tags, meta, map[string]any{"control": value}, tm)
	if err == nil {
		m.AddTag("method", "PUT")
	}
	return m, err
}

func (m *ccMessage) IsControl() bool {
	if !m.hasStringField("control") {
		return false
	}
	if method, ok := m.GetTag("method"); ok {
		return method == "PUT" || method == "GET"
	}
	return false
}

func (m *ccMessage) GetControlValue() (string, bool) {
	if m.IsControl() {
		if v, ok := m.GetField("control"); ok {
			return v.(string), true
		}
	}
	return "", false
}

func (m *ccMessage) GetControlMethod() (string, bool) {
	if m.IsControl() {
		if v, ok := m.GetTag("method"); ok {
			return v, true
		}
	}
	return "", false
}
