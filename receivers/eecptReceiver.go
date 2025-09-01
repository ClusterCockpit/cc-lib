// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package receivers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"
	influx "github.com/influxdata/line-protocol/v2/lineprotocol"
)

const CCCPT_RECEIVER_PORT = "8080"

type EECPTReceiverConfig struct {
	defaultReceiverConfig
	Addr string `json:"address"`
	Port string `json:"port"`
	Path string `json:"path"`

	// Maximum amount of time to wait for the next request when keep-alives are enabled
	// should be larger than the measurement interval to keep the connection open
	IdleTimeout string `json:"idle_timeout"`
	idleTimeout time.Duration

	// Controls whether HTTP keep-alives are enabled. By default, keep-alives are enabled
	KeepAlivesEnabled bool `json:"keep_alives_enabled"`

	// Basic authentication
	Username     string `json:"username"`
	Password     string `json:"password"`
	useBasicAuth bool

	AnalysisBufferLength int    `json:"buffer_size"`
	AnalysisInterval     string `json:"analysis_interval"`
	analysisInterval     time.Duration
}

type EECPTReceiverTask struct {
	numDistsFilled int
	dists          []struct {
		useful        float64
		usefulPlusMpi float64
	}
	distsLock sync.RWMutex
	subtasks  []*EECPTReceiverTask
}

type EECPTReceiverJob struct {
	jobLock sync.RWMutex
	tasks   map[int64]*EECPTReceiverTask
}

type EECPTReceiver struct {
	receiver
	// meta   map[string]string
	config          EECPTReceiverConfig
	server          *http.Server
	wg              sync.WaitGroup
	referenceBuffer []interface{}
	newBuffer       []interface{}
	bufferLock      sync.Mutex
	jobs            map[string]*EECPTReceiverJob
	analysisTicker  *time.Ticker
	analysisDone    chan bool
}

const chiSquareDistThreshold float64 = 1.0e-12

func GetChiSquareLimit(length int) float64 {
	switch {
	case length < 3:
		return 3.8145
	case length < 5:
		return 7.8147
	case length < 9:
		return 1.4067e1
	case length < 17:
		return 2.4996e1
	case length < 33:
		return 4.4985e1
	case length < 65:
		return 8.2529e1
	case length < 73:
		return 9.1670e1
	case length < 129:
		return 1.5430e2
	case length < 257:
		return 2.9325e2
	case length < 513:
		return 5.6470e2
	}
	return 1.0985e3
}

func (task *EECPTReceiverTask) averages() (float64, float64) {
	avgUseful := float64(0.0)
	avgUsefulPlusMpi := float64(0.0)
	task.distsLock.RLock()
	for _, dist := range task.dists {
		avgUseful += dist.useful
		avgUsefulPlusMpi += dist.usefulPlusMpi
	}
	task.distsLock.RUnlock()
	avgUseful /= float64(task.numDistsFilled)
	avgUsefulPlusMpi /= float64(task.numDistsFilled)
	return avgUseful, avgUsefulPlusMpi
}

func (task *EECPTReceiverTask) reset() {
	task.distsLock.RLock()
	task.dists = task.dists[:0]
	task.distsLock.RUnlock()
}

func (task *EECPTReceiverTask) add(useful, usefulPlusMpi float64) {
	task.distsLock.Lock()
	task.dists = append(task.dists, struct {
		useful        float64
		usefulPlusMpi float64
	}{
		useful:        useful,
		usefulPlusMpi: usefulPlusMpi,
	})
	task.distsLock.Unlock()
}

func (job *EECPTReceiverJob) chiSquareAnalysis() bool {
	result := float64(0.0)
	for t := range job.tasks {
		task := job.tasks[t]
		avgUseful, avgUsefulPlusMpi := task.averages()

		if avgUseful > chiSquareDistThreshold {
			x := task.dists[task.numDistsFilled-1].useful - avgUseful
			result += x * (x / avgUseful)
		}
		if avgUsefulPlusMpi > chiSquareDistThreshold {
			x := task.dists[task.numDistsFilled-1].usefulPlusMpi - avgUsefulPlusMpi
			result += x * (x / avgUsefulPlusMpi)
		}
	}
	return result > GetChiSquareLimit(len(job.tasks))
}

func (job *EECPTReceiverJob) newTask(id int64) {
	job.jobLock.Lock()
	if _, ok := job.tasks[id]; !ok {
		job.tasks[id] = new(EECPTReceiverTask)
		job.tasks[id].dists = make([]struct {
			useful        float64
			usefulPlusMpi float64
		}, 0)
	}
}

func (r *EECPTReceiver) Init(name string, config json.RawMessage) error {
	r.name = fmt.Sprintf("EECPTReceiver(%s)", name)

	// Set default values
	r.config.Port = HTTP_RECEIVER_PORT
	r.config.KeepAlivesEnabled = true
	// should be larger than the measurement interval to keep the connection open
	r.config.IdleTimeout = "120s"
	r.config.AnalysisBufferLength = 128

	// Read config
	if len(config) > 0 {
		err := json.Unmarshal(config, &r.config)
		if err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return err
		}
	}
	if len(r.config.Port) == 0 {
		return errors.New("not all configuration variables set required by EECPTReceiver")
	}

	// Check idle timeout config
	if len(r.config.IdleTimeout) > 0 {
		t, err := time.ParseDuration(r.config.IdleTimeout)
		if err == nil {
			cclog.ComponentDebug(r.name, "idleTimeout", t)
			r.config.idleTimeout = t
		}
	}
	// Check analysis interval config
	if len(r.config.AnalysisInterval) > 0 {
		t, err := time.ParseDuration(r.config.AnalysisInterval)
		if err == nil {
			cclog.ComponentDebug(r.name, "analysisInterval", t)
			r.config.analysisInterval = t
		}
	}

	// Check basic authentication config
	if len(r.config.Username) > 0 || len(r.config.Password) > 0 {
		r.config.useBasicAuth = true
	}
	if r.config.useBasicAuth && len(r.config.Username) == 0 {
		return errors.New("basic authentication requires username")
	}
	if r.config.useBasicAuth && len(r.config.Password) == 0 {
		return errors.New("basic authentication requires password")
	}

	// Check size of analysis buffer
	if r.config.AnalysisBufferLength <= 0 {
		return fmt.Errorf("buffer length of %d not allowed", r.config.AnalysisBufferLength)
	}
	//SetChiSquareLimit(r.config.AnalysisBufferLength)

	// Configure message processor
	msgp, err := mp.NewMessageProcessor()
	if err != nil {
		return fmt.Errorf("initialization of message processor failed: %v", err.Error())
	}
	r.mp = msgp
	if len(r.config.MessageProcessor) > 0 {
		err = r.mp.FromConfigJSON(r.config.MessageProcessor)
		if err != nil {
			return fmt.Errorf("failed parsing JSON for message processor: %v", err.Error())
		}
	}
	r.mp.AddAddMetaByCondition("true", "source", r.name)

	r.bufferLock.Lock()
	r.referenceBuffer = make([]interface{}, r.config.AnalysisBufferLength)
	r.newBuffer = make([]interface{}, r.config.AnalysisBufferLength)
	r.bufferLock.Unlock()

	//r.meta = map[string]string{"source": r.name}
	p := r.config.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	addr := fmt.Sprintf("%s:%s", r.config.Addr, r.config.Port)
	uri := addr + p
	cclog.ComponentDebug(r.name, "INIT", "listen on:", uri)

	// Register handler function r.ServerHttp for path p in the DefaultServeMux
	http.HandleFunc(p, r.ServerHttp)

	// Create http server
	r.server = &http.Server{
		Addr:        addr,
		Handler:     nil, // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout: r.config.idleTimeout,
	}
	r.server.SetKeepAlivesEnabled(r.config.KeepAlivesEnabled)

	return nil
}

func (r *EECPTReceiver) Start() {
	cclog.ComponentDebug(r.name, "START")
	r.wg.Add(1)
	go func() {
		err := r.server.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			cclog.ComponentError(r.name, err.Error())
		}
		r.wg.Done()
	}()
	r.analysisTicker = time.NewTicker(r.config.analysisInterval)
	r.analysisDone = make(chan bool)
	r.wg.Add(1)
	go func() {
		for {
			select {
			case <-r.analysisDone:
				r.analysisTicker.Stop()
				r.wg.Done()
				return
			case <-r.analysisTicker.C:
				for _, job := range r.jobs {
					same := job.chiSquareAnalysis()
					if same {
						// new region
						y, err := lp.NewEvent("region", map[string]string{"type": "node", "stype": "application"}, nil, "region changed", time.Now())
						if err == nil {
							m, err := r.mp.ProcessMessage(y)
							if err == nil && m != nil {
								r.sink <- m
							}
						}
					} else {
						r.referenceBuffer = r.referenceBuffer[:0]
						r.referenceBuffer = append(r.referenceBuffer, r.newBuffer...)
						r.newBuffer = r.newBuffer[:0]
					}
				}
			}
		}
	}()
}

func (r *EECPTReceiver) ServerHttp(w http.ResponseWriter, req *http.Request) {
	// Check request method, only post method is handled
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check basic authentication
	if r.config.useBasicAuth {
		username, password, ok := req.BasicAuth()
		if !ok || username != r.config.Username || password != r.config.Password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	if r.sink != nil {
		d := influx.NewDecoder(req.Body)
		for d.Next() {

			// Decode measurement name
			measurement, err := d.Measurement()
			if err != nil {
				msg := "ServerHttp: Failed to decode measurement: " + err.Error()
				cclog.ComponentError(r.name, msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			// Decode tags
			tags := make(map[string]string)
			for {
				key, value, err := d.NextTag()
				if err != nil {
					msg := "ServerHttp: Failed to decode tag: " + err.Error()
					cclog.ComponentError(r.name, msg)
					http.Error(w, msg, http.StatusInternalServerError)
					return
				}
				if key == nil {
					break
				}
				tags[string(key)] = string(value)
			}

			// Decode fields
			fields := make(map[string]interface{})
			for {
				key, value, err := d.NextField()
				if err != nil {
					msg := "ServerHttp: Failed to decode field: " + err.Error()
					cclog.ComponentError(r.name, msg)
					http.Error(w, msg, http.StatusInternalServerError)
					return
				}
				if key == nil {
					break
				}
				fields[string(key)] = value.Interface()
			}

			// Decode time stamp
			t, err := d.Time(influx.Nanosecond, time.Time{})
			if err != nil {
				msg := "ServerHttp: Failed to decode time stamp: " + err.Error()
				cclog.ComponentError(r.name, msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}

			y, _ := lp.NewMessage(
				string(measurement),
				tags,
				nil,
				fields,
				t,
			)

			if appl, ok := tags["application"]; ok {
				if _, ok := r.jobs[appl]; !ok {
					r.jobs[appl] = new(EECPTReceiverJob)
				}
				job := r.jobs[appl]

				pid := int64(0)
				if r, ok := fields["rank"]; ok {
					switch rank := r.(type) {
					case int, int32, int64:
						pid = rank.(int64)
					}
				}
				if _, ok := job.tasks[pid]; !ok {
					job.newTask(pid)
				}

				task := job.tasks[pid]

				job.bufferLock.Lock()
				job.newBuffer = append(r.newBuffer, y)
				if len(job.newBuffer) > r.config.AnalysisBufferLength {
					job.newBuffer = job.newBuffer[1:]
				}
				job.bufferLock.Unlock()
			}
			if r.sink != nil {
				m, err := r.mp.ProcessMessage(y)
				if err == nil && m != nil {
					r.sink <- m
				}
			}
		}
		// Check for IO errors
		err := d.Err()
		if err != nil {
			msg := "ServerHttp: Failed to decode: " + err.Error()
			cclog.ComponentError(r.name, msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (r *EECPTReceiver) Close() {
	if r.analysisDone != nil {
		r.analysisDone <- true
	}
	r.server.Shutdown(context.Background())
}

func NewEECPTReceiver(name string, config json.RawMessage) (Receiver, error) {
	r := new(EECPTReceiver)
	err := r.Init(name, config)
	return r, err
}
