//go:build !linux

// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package receivers

import "encoding/json"

// Map of all available receivers
var AvailableReceivers = map[string]func(name string, config json.RawMessage) (Receiver, error){
	"http":       NewHttpReceiver,
	"nats":       NewNatsReceiver,
	"eecpt":      NewEECPTReceiver,
	"prometheus": NewPrometheusReceiver,
}
