// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package receivers

import (
	"encoding/json"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"
)

// defaultReceiverConfig contains common configuration fields for all receivers.
type defaultReceiverConfig struct {
	Type             string          `json:"type"`                       // Receiver type identifier
	MessageProcessor json.RawMessage `json:"process_messages,omitempty"` // Optional message processing rules
}

// ReceiverConfig is the legacy configuration structure for receivers.
// Deprecated: Most receivers now use type-specific configuration structs.
type ReceiverConfig struct {
	Addr         string `json:"address"`                // Network address to bind/connect
	Port         string `json:"port"`                   // Network port
	Database     string `json:"database"`               // Database name (if applicable)
	Organization string `json:"organization,omitempty"` // Organization identifier
	Type         string `json:"type"`                   // Receiver type
}

type receiver struct {
	name string
	sink chan lp.CCMessage
	mp   mp.MessageProcessor
}

// Receiver is the interface all metric receivers must implement.
// Receivers collect metrics from various sources and send them to a sink channel.
type Receiver interface {
	Start()                         // Start begins the metric collection process
	Close()                         // Close stops the receiver and releases resources
	Name() string                   // Name returns the receiver's identifier
	SetSink(sink chan lp.CCMessage) // SetSink configures the output channel for collected metrics
}

// Name returns the name of the metric receiver
func (r *receiver) Name() string {
	return r.name
}

// SetSink set the sink channel
func (r *receiver) SetSink(sink chan lp.CCMessage) {
	r.sink = sink
}
