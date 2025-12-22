// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"time"
)

// NewLog creates a new log message.
// Log messages are used to transmit textual log data through the ClusterCockpit messaging system.
//
// Parameters:
//   - name: The name/category of the log message (e.g., "system_log", "application_log")
//   - tags: Optional tags for categorizing the log message (e.g., severity, source)
//   - meta: Optional metadata information
//   - log: The log message content as a string
//   - tm: Timestamp when the log message was generated
//
// Returns a CCMessage with the "log" field set to the provided log content.
func NewLog(name string,
	tags map[string]string,
	meta map[string]string,
	log string,
	tm time.Time,
) (CCMessage, error) {
	return NewMessage(name, tags, meta, map[string]any{"log": log}, tm)
}

func (m *ccMessage) IsLog() bool {
	if v, ok := m.GetField("log"); ok {
		if _, ok := v.(string); ok {
			return true
		}
	}
	return false
}

func (m *ccMessage) GetLogValue() (string, bool) {
	if m.IsLog() {
		if v, ok := m.GetField("log"); ok {
			return v.(string), true
		}
	}
	return "", false
}
