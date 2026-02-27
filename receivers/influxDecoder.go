// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

// Package receivers provides a modular system for collecting metrics, events, and logs from various sources.
// It defines a common Receiver interface and a ReceiveManager to orchestrate multiple receiver instances.
// Receivers collect data from external sources, convert it into CCMessage format, and send it to a unified sink channel.
package receivers

import (
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	influx "github.com/ClusterCockpit/cc-line-protocol/v2/lineprotocol"
)

// DecodeInfluxMessage decodes a single InfluxDB line protocol message from the decoder
// Returns the decoded CCMessage or an error if decoding fails
func DecodeInfluxMessage(d *influx.Decoder) (lp.CCMessage, error) {
	measurement, err := d.Measurement()
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for {
		key, value, err := d.NextTag()
		if err != nil {
			return nil, err
		}
		if key == nil {
			break
		}
		tags[string(key)] = string(value)
	}

	fields := make(map[string]any)
	for {
		key, value, err := d.NextField()
		if err != nil {
			return nil, err
		}
		if key == nil {
			break
		}
		fields[string(key)] = value.Interface()
	}

	t, err := d.Time(influx.Nanosecond, time.Time{})
	if err != nil {
		return nil, err
	}

	return lp.NewMessage(
		string(measurement),
		tags,
		nil,
		fields,
		t,
	)
}
