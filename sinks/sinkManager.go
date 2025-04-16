// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package sinks

import (
	"encoding/json"
	"fmt"
	"sync"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
)

const SINK_MAX_FORWARD = 50

type Sink interface {
	Write(point lp.CCMessage) error // Write metric to the sink
	Flush() error                   // Flush buffered metrics
	Close()                         // Close / finish metric sink
	Name() string                   // Name of the metric sink
}

// Sink manager access functions
type SinkManager interface {
	Init(wg *sync.WaitGroup, sinkConfig json.RawMessage) error
	AddInput(input chan lp.CCMessage)
	AddOutput(name string, config json.RawMessage) error
	Start()
	Close()
}

// Metric collector manager data structure
type sinkManager struct {
	input      chan lp.CCMessage // input channel
	done       chan bool         // channel to finish / stop metric sink manager
	wg         *sync.WaitGroup   // wait group for all goroutines in cc-metric-collector
	sinks      map[string]Sink   // Mapping sink name to sink
	maxForward int               // number of metrics to write maximally in one iteration
}

// Init initializes the sink manager by:
// * Reading its configuration file
// * Adding the configured sinks and providing them with the corresponding config
func (sm *sinkManager) Init(wg *sync.WaitGroup, sinkConfig json.RawMessage) error {
	sm.input = nil
	sm.done = make(chan bool)
	sm.wg = wg
	sm.sinks = make(map[string]Sink, 0)
	sm.maxForward = SINK_MAX_FORWARD

	// Parse config
	var rawConfigs map[string]json.RawMessage
	err := json.Unmarshal(sinkConfig, (&rawConfigs))
	if err != nil {
		cclog.ComponentError("SinkManager", err.Error())
		return err
	}

	// Start sinks
	for name, raw := range rawConfigs {
		err = sm.AddOutput(name, raw)
		if err != nil {
			cclog.ComponentError("SinkManager", err)
			continue
		}
	}

	// Check that at least one sink is running
	if len(sm.sinks) <= 0 {
		cclog.ComponentError("SinkManager", "Found no usable sinks")
		return fmt.Errorf("found no usable sinks")
	}

	return nil
}

// Start starts the sink managers background task, which
// distributes received metrics to the sinks
func (sm *sinkManager) Start() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()

		// Sink manager is done
		done := func() {
			for _, s := range sm.sinks {
				s.Close()
			}

			close(sm.done)
			cclog.ComponentDebug("SinkManager", "DONE")
		}

		toTheSinks := func(p lp.CCMessage) {
			// Send received metric to all outputs
			cclog.ComponentDebug("SinkManager", "WRITE", p)
			for _, s := range sm.sinks {
				if err := s.Write(p); err != nil {
					cclog.ComponentError("SinkManager", "WRITE", s.Name(), "write failed:", err.Error())
				}
			}
		}

		for {
			select {
			case <-sm.done:
				done()
				return

			case p := <-sm.input:
				toTheSinks(p)
				for i := 0; len(sm.input) > 0 && i < sm.maxForward; i++ {
					p := <-sm.input
					toTheSinks(p)
				}
			}
		}
	}()

	// Sink manager is started
	cclog.ComponentDebug("SinkManager", "STARTED")
}

// AddInput adds the input channel to the sink manager
func (sm *sinkManager) AddInput(input chan lp.CCMessage) {
	sm.input = input
}

func (sm *sinkManager) AddOutput(name string, rawConfig json.RawMessage) error {
	var err error
	var sinkConfig defaultSinkConfig
	if len(rawConfig) > 0 {
		err := json.Unmarshal(rawConfig, &sinkConfig)
		if err != nil {
			return err
		}
	}
	if _, found := AvailableSinks[sinkConfig.Type]; !found {
		cclog.ComponentError("SinkManager", "SKIP", name, "unknown sink:", sinkConfig.Type)
		return err
	}
	s, err := AvailableSinks[sinkConfig.Type](name, rawConfig)
	if err != nil {
		cclog.ComponentError("SinkManager", "SKIP", name, "initialization failed:", err.Error())
		return err
	}
	sm.sinks[name] = s
	cclog.ComponentDebug("SinkManager", "ADD SINK", s.Name(), "with name", fmt.Sprintf("'%s'", name))
	return nil
}

// Close finishes / stops the sink manager
func (sm *sinkManager) Close() {
	cclog.ComponentDebug("SinkManager", "CLOSE")
	sm.done <- true
	// wait for close of channel sm.done
	<-sm.done
}

// New creates a new initialized sink manager
func New(wg *sync.WaitGroup, sinkConfig json.RawMessage) (SinkManager, error) {
	sm := new(sinkManager)
	err := sm.Init(wg, sinkConfig)
	if err != nil {
		return nil, err
	}
	return sm, err
}
