//go:build linux

// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package sinks

import "encoding/json"

// Map of all available sinks
var AvailableSinks = map[string]func(name string, config json.RawMessage) (Sink, error){
	"ganglia":     NewGangliaSink,
	"stdout":      NewStdoutSink,
	"nats":        NewNatsSink,
	"influxdb":    NewInfluxSink,
	"influxasync": NewInfluxAsyncSink,
	"http":        NewHttpSink,
	"prometheus":  NewPrometheusSink,
}
