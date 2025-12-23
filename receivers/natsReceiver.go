// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package receivers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"
	influx "github.com/influxdata/line-protocol/v2/lineprotocol"
	nats "github.com/nats-io/nats.go"
)

// NatsReceiverConfig configures the NATS receiver for subscribing to metric messages.
type NatsReceiverConfig struct {
	defaultReceiverConfig
	Addr     string `json:"address"`             // NATS server address (default: localhost)
	Port     string `json:"port"`                // NATS server port (default: 4222)
	Subject  string `json:"subject"`             // NATS subject to subscribe to (required)
	User     string `json:"user,omitempty"`      // Username for authentication
	Password string `json:"password,omitempty"`  // Password for authentication
	NkeyFile string `json:"nkey_file,omitempty"` // Path to NKey credentials file
}

type NatsReceiver struct {
	receiver
	nc  *nats.Conn
	sub *nats.Subscription
	// meta   map[string]string
	config NatsReceiverConfig
}

// Start subscribes to the configured NATS subject
// Messages wil be handled by r._NatsReceive
func (r *NatsReceiver) Start() {
	cclog.ComponentDebug(r.name, "START")
	sub, err := r.nc.Subscribe(r.config.Subject, r._NatsReceive)
	if err != nil {
		msg := fmt.Sprintf("Failed to subscribe to subject '%s': %s", r.config.Subject, err.Error())
		cclog.ComponentError(r.name, msg)
	}
	r.sub = sub
}

// _NatsReceive receives subscribed messages from the NATS server
func (r *NatsReceiver) _NatsReceive(m *nats.Msg) {
	if r.sink == nil {
		return
	}

	d := influx.NewDecoderWithBytes(m.Data)
	for d.Next() {
		y, err := DecodeInfluxMessage(d)
		if err != nil {
			cclog.ComponentError(r.name, "_NatsReceive: Failed to decode message:", err)
			return
		}

		msg, err := r.mp.ProcessMessage(y)
		if err == nil && msg != nil {
			r.sink <- msg
		}
	}
}

// Close closes the connection to the NATS server
func (r *NatsReceiver) Close() {
	if r.nc == nil {
		return
	}

	defer r.nc.Close()
	defer r.sub.Unsubscribe()

	cclog.ComponentDebug(r.name, "DRAIN")
	err := r.sub.Drain()
	if err != nil {
		cclog.ComponentError(r.name, "Failed to drain subscription to subject", r.config.Subject, ":", err)
	}
	cclog.ComponentDebug(r.name, "CLOSE")
}

// NewNatsReceiver creates a new Receiver which subscribes to messages from a NATS server
func NewNatsReceiver(name string, config json.RawMessage) (Receiver, error) {
	var uinfo nats.Option = nil
	r := new(NatsReceiver)
	r.name = fmt.Sprintf("NatsReceiver(%s)", name)

	// Read configuration file, allow overwriting default config
	r.config.Addr = "localhost"
	r.config.Port = "4222"
	if len(config) > 0 {
		err := json.Unmarshal(config, &r.config)
		if err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return nil, err
		}
	}
	if len(r.config.Addr) == 0 ||
		len(r.config.Port) == 0 ||
		len(r.config.Subject) == 0 {
		return nil, errors.New("not all configuration variables set required by NatsReceiver")
	}
	p, err := mp.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("initialization of message processor failed: %w", err)
	}
	r.mp = p
	if len(r.config.MessageProcessor) > 0 {
		err = r.mp.FromConfigJSON(r.config.MessageProcessor)
		if err != nil {
			return nil, fmt.Errorf("failed parsing JSON for message processor: %w", err)
		}
	}

	// Set metadata
	// r.meta = map[string]string{
	// 	"source": r.name,
	// }
	r.mp.AddAddMetaByCondition("true", "source", r.name)

	if len(r.config.User) > 0 && len(r.config.Password) > 0 {
		uinfo = nats.UserInfo(r.config.User, r.config.Password)
	} else if len(r.config.NkeyFile) > 0 {
		_, err := os.Stat(r.config.NkeyFile)
		if err == nil {
			uinfo = nats.UserCredentials(r.config.NkeyFile)
		} else {
			cclog.ComponentError(r.name, "NKEY file", r.config.NkeyFile, "does not exist: %v", err.Error())
			return nil, err
		}
	}

	// Connect to NATS server
	url := fmt.Sprintf("nats://%s:%s", r.config.Addr, r.config.Port)
	cclog.ComponentDebug(r.name, "NewNatsReceiver ", url, " Subject ", r.config.Subject)
	if nc, err := nats.Connect(url, uinfo); err == nil {
		r.nc = nc
	} else {
		r.nc = nil
		return nil, err
	}

	sub, err := r.nc.Subscribe(r.config.Subject, func(m *nats.Msg) {})
	if err != nil {
		err = fmt.Errorf("Failed to test subscribe to subject '%s': %s", r.config.Subject, err.Error())
		cclog.ComponentError(r.name, err)
		return nil, err
	}
	sub.Unsubscribe()

	return r, nil
}
