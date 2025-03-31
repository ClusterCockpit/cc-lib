// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"reflect"
	"time"
)

func NewGetControl(name string,
	tags map[string]string,
	meta map[string]string,
	tm time.Time,
) (CCMessage, error) {
	m, err := NewMessage(name, tags, meta, map[string]interface{}{"control": ""}, tm)
	if err == nil {
		m.AddTag("method", "GET")
	}
	return m, err
}

func NewPutControl(name string,
	tags map[string]string,
	meta map[string]string,
	value string,
	tm time.Time,
) (CCMessage, error) {
	m, err := NewMessage(name, tags, meta, map[string]interface{}{"control": value}, tm)
	if err == nil {
		m.AddTag("method", "PUT")
	}
	return m, err
}

func (m *ccMessage) IsControl() bool {
	if v, ok := m.GetField("control"); ok {
		if me, ok := m.GetTag("method"); ok {
			if reflect.TypeOf(v) == reflect.TypeOf("string") && (me == "PUT" || me == "GET") {
				return true
			}
		}
	}
	return false
}

func (m *ccMessage) GetControlValue() string {
	if m.IsControl() {
		if v, ok := m.GetField("control"); ok {
			return v.(string)
		}
	}
	return ""
}

func (m *ccMessage) GetControlMethod() string {
	if m.IsControl() {
		if v, ok := m.GetTag("method"); ok {
			return v
		}
	}
	return ""
}
