// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package receivers

import (
	"encoding/json"
	"fmt"
	"sync"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
)

type receiveManager struct {
	inputs []Receiver
	output chan lp.CCMessage
	config []json.RawMessage
}

type ReceiveManager interface {
	Init(wg *sync.WaitGroup, receiverConfig json.RawMessage) error
	AddInput(name string, rawConfig json.RawMessage) error
	AddOutput(output chan lp.CCMessage)
	Start()
	Close()
}

func (rm *receiveManager) Init(wg *sync.WaitGroup, receiverConfig json.RawMessage) error {
	// Initialize struct fields
	rm.inputs = make([]Receiver, 0)
	rm.output = nil
	rm.config = make([]json.RawMessage, 0)

	// Parse config
	var rawConfigs map[string]json.RawMessage
	err := json.Unmarshal(receiverConfig, (&rawConfigs))
	if err != nil {
		cclog.ComponentError("ReceiveManager", err.Error())
		return err
	}

	// Start receivers
	for name, raw := range rawConfigs {
		err = rm.AddInput(name, raw)
		if err != nil {
			cclog.ComponentError("ReceiveManager", err)
			continue
		}
	}

	return nil
}

func (rm *receiveManager) Start() {
	cclog.ComponentDebug("ReceiveManager", "START")

	for _, r := range rm.inputs {
		cclog.ComponentDebug("ReceiveManager", "START", r.Name())
		r.Start()
	}
	cclog.ComponentDebug("ReceiveManager", "STARTED")
}

func (rm *receiveManager) AddInput(name string, rawConfig json.RawMessage) error {
	var config defaultReceiverConfig
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		cclog.ComponentError("ReceiveManager", "SKIP", config.Type, "JSON config error:", err.Error())
		return err
	}
	if config.Type == "" {
		cclog.ComponentError("ReceiveManager", "SKIP", "JSON config for receiver", name, "does not contain a receiver type")
		return fmt.Errorf("JSON config for receiver %s does not contain a receiver type", name)
	}
	if _, found := AvailableReceivers[config.Type]; !found {
		cclog.ComponentError("ReceiveManager", "SKIP", "unknown receiver type:", config.Type)
		return fmt.Errorf("unknown receiver type: %s", config.Type)
	}
	r, err := AvailableReceivers[config.Type](name, rawConfig)
	if err != nil {
		cclog.ComponentError("ReceiveManager", "SKIP", name, "initialization failed:", err.Error())
		return err
	}
	rm.inputs = append(rm.inputs, r)
	rm.config = append(rm.config, rawConfig)
	cclog.ComponentDebug("ReceiveManager", "ADD RECEIVER", r.Name())
	return nil
}

func (rm *receiveManager) AddOutput(output chan lp.CCMessage) {
	rm.output = output
	for _, r := range rm.inputs {
		r.SetSink(rm.output)
	}
}

func (rm *receiveManager) Close() {
	cclog.ComponentDebug("ReceiveManager", "CLOSE")

	// Close all receivers
	for _, r := range rm.inputs {
		cclog.ComponentDebug("ReceiveManager", "CLOSE", r.Name())
		r.Close()
	}

	cclog.ComponentDebug("ReceiveManager", "DONE")
}

func New(wg *sync.WaitGroup, receiverConfig json.RawMessage) (ReceiveManager, error) {
	r := new(receiveManager)
	err := r.Init(wg, receiverConfig)
	if err != nil {
		return nil, err
	}
	return r, err
}
