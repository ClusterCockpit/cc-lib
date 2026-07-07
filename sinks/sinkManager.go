// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)
package sinks

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

const SINK_MAX_FORWARD = 50

// Minimum interval between warnings about a full sink queue
const SINK_DROP_WARN_INTERVAL = 5 * time.Second

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
	input      chan lp.CCMessage            // input channel
	done       chan bool                    // channel to finish / stop metric sink manager
	wg         *sync.WaitGroup              // wait group for all goroutines in cc-metric-collector
	sinks      map[string]Sink              // Mapping sink name to sink
	queues     map[string]chan lp.CCMessage // per-sink queue, so a slow sink does not stall the others
	writer_wg  sync.WaitGroup               // wait group for the per-sink writer goroutines
	maxForward int                          // number of metrics to write maximally in one iteration
}

// Init initializes the sink manager by:
// * Reading its configuration file
// * Adding the configured sinks and providing them with the corresponding config
func (sm *sinkManager) Init(wg *sync.WaitGroup, sinkConfig json.RawMessage) error {
	sm.input = nil
	sm.done = make(chan bool)
	sm.wg = wg
	sm.sinks = make(map[string]Sink, 0)
	sm.queues = make(map[string]chan lp.CCMessage)
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
	// One writer goroutine per sink, so a slow sink only fills its own queue
	// instead of stalling the other sinks and the whole metric pipeline
	for name, s := range sm.sinks {
		queue := sm.queues[name]
		sm.writer_wg.Go(func() {
			for p := range queue {
				if err := s.Write(p); err != nil {
					cclog.ComponentError("SinkManager", "WRITE", s.Name(), "write failed:", err.Error())
				}
			}
		})
	}

	sm.wg.Go(func() {
		// Sink manager is done
		done := func() {
			// Close the queues and wait for the writers to drain them,
			// then close the sinks to flush their buffered metrics
			for _, q := range sm.queues {
				close(q)
			}
			sm.writer_wg.Wait()
			for _, s := range sm.sinks {
				s.Close()
			}

			close(sm.done)
			cclog.ComponentDebug("SinkManager", "DONE")
		}

		dropped := make(map[string]int)
		var lastDropWarn time.Time

		toTheSinks := func(p lp.CCMessage) {
			// Send received metric to all outputs
			cclog.ComponentDebug("SinkManager", "WRITE", p)
			for name, q := range sm.queues {
				select {
				case q <- p:
				default:
					dropped[name]++
				}
			}
			if len(dropped) > 0 && time.Since(lastDropWarn) >= SINK_DROP_WARN_INTERVAL {
				for name, n := range dropped {
					cclog.ComponentWarn("SinkManager", "sink", name, "is too slow, queue full,", n, "messages dropped")
				}
				clear(dropped)
				lastDropWarn = time.Now()
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
	})

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
	// Queue between the sink manager and the sink's writer goroutine, sized to
	// hold one interval's burst of per-hwthread metrics on nodes with many cores
	queueLength := sinkConfig.QueueLength
	if queueLength <= 0 {
		queueLength = max(4096, 24*runtime.NumCPU())
	}
	sm.queues[name] = make(chan lp.CCMessage, queueLength)
	cclog.ComponentDebug("SinkManager", "ADD SINK", s.Name(), "with name", fmt.Sprintf("'%s'", name), "queue length", queueLength)
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
