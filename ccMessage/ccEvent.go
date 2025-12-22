// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"time"
)

// NewEvent creates a new event message.
// Events represent significant occurrences in the ClusterCockpit system, such as job starts/stops,
// system state changes, or other notable incidents.
//
// Parameters:
//   - name: The name/type of the event (e.g., "start_job", "stop_job", "node_down")
//   - tags: Optional tags for categorizing the event
//   - meta: Optional metadata information
//   - event: The event payload as a string (can be JSON, plain text, or any other format)
//   - tm: Timestamp when the event occurred
//
// Returns a CCMessage with the "event" field set to the provided event payload.
func NewEvent(name string,
	tags map[string]string,
	meta map[string]string,
	event string,
	tm time.Time,
) (CCMessage, error) {
	return NewMessage(name, tags, meta, map[string]any{"event": event}, tm)
}

func (m *ccMessage) IsEvent() bool {
	if v, ok := m.GetField("event"); ok {
		if _, ok := v.(string); ok {
			return true
		}
	}
	return false
}

func (m *ccMessage) GetEventValue() (string, bool) {
	if m.IsEvent() {
		if v, ok := m.GetField("event"); ok {
			return v.(string), true
		}
	}
	return "", false
}
