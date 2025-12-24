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
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	mp "github.com/ClusterCockpit/cc-lib/v2/messageProcessor"
	influx "github.com/influxdata/line-protocol/v2/lineprotocol"
)

const (
	CCCPT_RECEIVER_PORT            = "8080"
	chiSquareDistThreshold float64 = 1.0e-12
)

// overwritten by configuration. 4 is the minimum
var eecpt_analysis_buffer_size = 4

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

	AnalysisBufferLength int    `json:"analysis_buffer_size"`
	AnalysisInterval     string `json:"analysis_interval"`
	AnalysisMetric       string `json:"analysis_metric"`
	analysisInterval     time.Duration
}

type EECPTReceiverTask struct {
	ident      string
	tags       map[string]string
	buffer     []float64
	bufferLock sync.RWMutex
	subtasks   map[int64]*EECPTReceiverTask
}

type EECPTReceiverJob struct {
	ident string
	tags  map[string]string
	tasks map[int64]*EECPTReceiverTask
}

type EECPTReceiver struct {
	receiver
	// meta   map[string]string
	config         EECPTReceiverConfig
	server         *http.Server
	wg             sync.WaitGroup
	jobs           map[string]*EECPTReceiverJob
	analysisTicker *time.Ticker
	analysisDone   chan bool
}

func NewJob(ident string) *EECPTReceiverJob {
	j := new(EECPTReceiverJob)
	j.ident = ident
	cclog.ComponentDebug("EECPTReceiver", "New job: ", ident)
	j.tags = make(map[string]string)
	j.tasks = make(map[int64]*EECPTReceiverTask)
	return j
}

func (job *EECPTReceiverJob) newTask(id int64) {
	if _, ok := job.tasks[id]; !ok {
		job.tasks[id] = new(EECPTReceiverTask)
		job.tasks[id].ident = fmt.Sprintf("%d", id)
		cclog.ComponentDebug("EECPTReceiver", "New task: ", id)
		job.tasks[id].buffer = make([]float64, 0)
		job.tasks[id].tags = make(map[string]string)
		job.tasks[id].subtasks = make(map[int64]*EECPTReceiverTask)
	}
}

// func (task *EECPTReceiverTask) newSubTask(id int64) {
// 	if _, ok := task.subtasks[id]; !ok {
// 		task.subtasks[id] = new(EECPTReceiverTask)
// 		task.subtasks[id].ident = fmt.Sprintf("%d", id)
// 		task.subtasks[id].buffer = make([]float64, 0)
// 		task.subtasks[id].tags = make(map[string]string)
// 	}
// }

// Analyse performs chi-square statistical test to detect phase transitions in application behavior.
// It computes the chi-square statistic by comparing the expected rate of change (prev)
// with the observed change (last) across all tasks in the job.
// Returns the chi-square test statistic value.
func (job *EECPTReceiverJob) Analyse() float64 {
	result := float64(0)
	for _, task := range job.tasks {
		prev, last, err := task.Analyse()
		if err == nil && prev > chiSquareDistThreshold {
			result += math.Pow(last-prev, 2) / prev
		}
	}
	return result
}

func (job *EECPTReceiverJob) Reset() {
	for _, task := range job.tasks {
		task.Reset()
	}
}

// ChiSquareLimit returns the critical chi-square value at 95% confidence level (p=0.05)
// for the given number of degrees of freedom (number of tasks).
// These are pre-calculated chi-square distribution values used to determine
// if a phase transition is statistically significant.
func (job *EECPTReceiverJob) ChiSquareLimit() float64 {
	length := len(job.tasks)
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

// Analyse computes the expected rate of change (prev) and last observed change (last)
// for this task's metric buffer. Returns (prev, last, error).
// prev = average rate of change over buffer history
// last = most recent change
func (task *EECPTReceiverTask) Analyse() (float64, float64, error) {
	task.bufferLock.RLock()
	defer task.bufferLock.RUnlock()
	buflen := len(task.buffer)
	if buflen < 3 {
		return 0, 0, fmt.Errorf("Analysis of task %s requires at least 3 entries in buffer but have only %d", task.ident, buflen)
	}
	prev := float64(task.buffer[buflen-2]-task.buffer[0]) / float64(buflen-2)
	last := float64(task.buffer[buflen-1] - task.buffer[buflen-2])
	return prev, last, nil
}

func (task *EECPTReceiverTask) PrintBuffer() {
	task.bufferLock.RLock()
	buflen := len(task.buffer)
	strbuf := make([]string, 0, buflen)
	for _, x := range task.buffer {
		strbuf = append(strbuf, fmt.Sprintf("%f", x))
	}
	fmt.Println(strings.Join(strbuf, ","))
	task.bufferLock.RUnlock()
}

func (task *EECPTReceiverTask) Add(value float64) {
	task.bufferLock.Lock()
	// Append new value to buffer
	task.buffer = append(task.buffer, value)
	// If the buffer has exceeded its configured size, drop the oldest value
	if len(task.buffer) > eecpt_analysis_buffer_size {
		task.buffer = task.buffer[1:]
	}
	task.bufferLock.Unlock()
}

func (task *EECPTReceiverTask) Reset() {
	task.bufferLock.Lock()
	// store the last value in the buffer
	last := task.buffer[len(task.buffer)-1]
	// reset buffer to zero entries
	task.buffer = task.buffer[:0]
	// add the last value back to the buffer
	task.buffer = append(task.buffer, last)
	task.bufferLock.Unlock()
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
	r.wg.Add(1)
	go func(myr **EECPTReceiver) {
		for {
			select {
			case <-(*myr).analysisDone:
				(*myr).analysisTicker.Stop()
				(*myr).wg.Done()
				return
			case <-(*myr).analysisTicker.C:
				for _, job := range (*myr).jobs {
					result := job.Analyse()
					if result > job.ChiSquareLimit() {
						cclog.ComponentDebug(r.name, fmt.Sprintf("Job %s changed phases (analysis %f chiSquareLimit %f)", job.ident, result, job.ChiSquareLimit()))
						y, err := lp.NewEvent("region", map[string]string{"type": "node", "stype": "application"}, nil, "region changed", time.Now())
						if err == nil {
							y.AddTag("stype-id", job.ident)
							m, err := (*myr).mp.ProcessMessage(y)
							if err == nil && m != nil {
								(*myr).sink <- m
							}
						}
						job.Reset()
					} else {
						cclog.ComponentDebug(r.name, fmt.Sprintf("Job %s no change (analysis %f chiSquareLimit %f)", job.ident, result, job.ChiSquareLimit()))
					}
				}
			}
		}
	}(&r)
}

func fieldToFloat64(input interface{}) float64 {
	switch in := input.(type) {
	case int:
		return float64(in)
	case int32:
		return float64(in)
	case int64:
		return float64(in)
	case uint:
		return float64(in)
	case uint32:
		return float64(in)
	case uint64:
		return float64(in)
	case float32:
		return float64(in)
	case float64:
		return in
	case string:
		x, err := strconv.ParseFloat(in, 64)
		if err == nil {
			return x
		}
	}
	return math.NaN()
}

func (r *EECPTReceiver) toAnalysis(msg lp.CCMessage) {
	jobid := ""
	if j, ok := msg.GetTag("jobid"); ok {
		jobid = j
	}
	if len(jobid) == 0 && msg.HasTag("application") {
		jobid, _ = msg.GetTag("application")
	}
	rank := int64(-1)
	if t, ok := msg.GetTag("rank"); ok {
		x, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			rank = x
		}
	} else {
		if t, ok := msg.GetField("rank"); ok {
			rank = int64(fieldToFloat64(t))
		}
	}
	if t, ok := msg.GetTag("pid"); ok {
		x, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			rank = x
		}
	} else {
		if t, ok := msg.GetField("pid"); ok {
			rank = int64(t.(int))
		}
	}
	value := float64(0)
	if v, ok := msg.GetField("value"); ok {
		value = fieldToFloat64(v)
	}
	if _, ok := r.jobs[jobid]; !ok {
		newjob := NewJob(jobid)
		r.jobs[jobid] = newjob
	}
	job := r.jobs[jobid]

	if _, ok := job.tasks[rank]; !ok {
		job.newTask(rank)
	}
	task := job.tasks[rank]

	task.Add(value)
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

	if r.sink == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	d := influx.NewDecoder(req.Body)
	for d.Next() {
		y, err := DecodeInfluxMessage(d)
		if err != nil {
			msg := "ServerHttp: Failed to decode message: " + err.Error()
			cclog.ComponentError(r.name, msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		m, err := r.mp.ProcessMessage(y)
		if err == nil && m != nil {
			r.sink <- m
			if m.Name() == r.config.AnalysisMetric {
				r.toAnalysis(m)
			}
		}
	}

	if err := d.Err(); err != nil {
		msg := "ServerHttp: Failed to decode: " + err.Error()
		cclog.ComponentError(r.name, msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
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
	r.name = fmt.Sprintf("EECPTReceiver(%s)", name)

	r.config.Port = HTTP_RECEIVER_PORT
	r.config.KeepAlivesEnabled = true
	r.config.IdleTimeout = "120s"
	r.config.AnalysisBufferLength = eecpt_analysis_buffer_size
	r.config.AnalysisInterval = "5m"
	r.config.AnalysisMetric = "region_metric"

	if len(config) > 0 {
		err := json.Unmarshal(config, &r.config)
		if err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return nil, err
		}
	}
	if len(r.config.Port) == 0 {
		return nil, errors.New("not all configuration variables set required by EECPTReceiver")
	}

	if len(r.config.IdleTimeout) > 0 {
		t, err := time.ParseDuration(r.config.IdleTimeout)
		if err == nil {
			cclog.ComponentDebug(r.name, "idleTimeout: ", t)
			r.config.idleTimeout = t
		}
	}

	if len(r.config.AnalysisInterval) > 0 {
		t, err := time.ParseDuration(r.config.AnalysisInterval)
		if err == nil {
			cclog.ComponentDebug(r.name, "analysisInterval: ", t)
			r.config.analysisInterval = t
		}
	}

	if len(r.config.Username) > 0 || len(r.config.Password) > 0 {
		r.config.useBasicAuth = true
	}
	if r.config.useBasicAuth && len(r.config.Username) == 0 {
		return nil, errors.New("basic authentication requires username")
	}
	if r.config.useBasicAuth && len(r.config.Password) == 0 {
		return nil, errors.New("basic authentication requires password")
	}

	if r.config.AnalysisBufferLength <= 0 {
		return nil, fmt.Errorf("buffer length of %d not allowed", r.config.AnalysisBufferLength)
	}
	eecpt_analysis_buffer_size = r.config.AnalysisBufferLength

	msgp, err := mp.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("initialization of message processor failed: %w", err)
	}
	r.mp = msgp
	if len(r.config.MessageProcessor) > 0 {
		err = r.mp.FromConfigJSON(r.config.MessageProcessor)
		if err != nil {
			return nil, fmt.Errorf("failed parsing JSON for message processor: %w", err)
		}
	}
	r.mp.AddAddMetaByCondition("true", "source", r.name)

	p := r.config.Path
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	addr := fmt.Sprintf("%s:%s", r.config.Addr, r.config.Port)
	uri := addr + p
	cclog.ComponentDebug(r.name, "INIT ", "listen on:", uri)

	http.HandleFunc(p, r.ServerHttp)

	r.server = &http.Server{
		Addr:        addr,
		Handler:     nil,
		IdleTimeout: r.config.idleTimeout,
	}
	r.server.SetKeepAlivesEnabled(r.config.KeepAlivesEnabled)

	r.jobs = make(map[string]*EECPTReceiverJob)
	r.analysisDone = make(chan bool)

	return r, nil
}
