// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"time"
)

// NewQuery creates a new CCMessage of type Query
func NewQuery(name string,
	tags map[string]string,
	meta map[string]string,
	q string,
	tm time.Time,
) (CCMessage, error) {
	return NewMessage(name, tags, meta, map[string]any{"query": q}, tm)
}

func (m *ccMessage) IsQuery() bool {
	return m.hasStringField("query")
}

// GetQueryValue returns the query string if the message is of type Query
func (m *ccMessage) GetQueryValue() (string, bool) {
	if m.IsQuery() {
		if v, ok := m.GetField("query"); ok {
			return v.(string), true
		}
	}
	return "", false
}
