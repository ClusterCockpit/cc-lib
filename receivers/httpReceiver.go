// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

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
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"
	influx "github.com/influxdata/line-protocol/v2/lineprotocol"
)

const HTTP_RECEIVER_PORT = "8080"

// HttpReceiverConfig configures the HTTP receiver for accepting metrics via POST requests.
type HttpReceiverConfig struct {
	defaultReceiverConfig
	Addr string `json:"address"` // Listen address (default: empty for all interfaces)
	Port string `json:"port"`    // Listen port (default: 8080)
	Path string `json:"path"`    // HTTP path to listen on

	IdleTimeout string `json:"idle_timeout"` // Max idle time for keep-alive connections (default: 120s)
	idleTimeout time.Duration

	KeepAlivesEnabled bool `json:"keep_alives_enabled"` // Enable HTTP keep-alive (default: true)

	Username     string `json:"username"` // Basic auth username (optional)
	Password     string `json:"password"` // Basic auth password (optional)
	useBasicAuth bool
}

type HttpReceiver struct {
	receiver
	// meta   map[string]string
	config HttpReceiverConfig
	server *http.Server
	wg     sync.WaitGroup
}

func (r *HttpReceiver) Start() {
	cclog.ComponentDebug(r.name, "START")
	r.wg.Add(1)
	go func() {
		err := r.server.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			cclog.ComponentError(r.name, err.Error())
		}
		r.wg.Done()
	}()
}

func (r *HttpReceiver) ServerHttp(w http.ResponseWriter, req *http.Request) {
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

func (r *HttpReceiver) Close() {
	cclog.ComponentDebug(r.name, "CLOSE")
	r.server.Shutdown(context.Background())
	r.wg.Wait()
	cclog.ComponentDebug(r.name, "DONE")
}

func NewHttpReceiver(name string, config json.RawMessage) (Receiver, error) {
	r := new(HttpReceiver)
	r.name = fmt.Sprintf("HttpReceiver(%s)", name)

	r.config.Port = HTTP_RECEIVER_PORT
	r.config.KeepAlivesEnabled = true
	r.config.IdleTimeout = "120s"

	if len(config) > 0 {
		err := json.Unmarshal(config, &r.config)
		if err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return nil, err
		}
	}
	if len(r.config.Port) == 0 {
		return nil, errors.New("not all configuration variables set required by HttpReceiver")
	}

	if len(r.config.IdleTimeout) > 0 {
		t, err := time.ParseDuration(r.config.IdleTimeout)
		if err == nil {
			cclog.ComponentDebug(r.name, "idleTimeout", t)
			r.config.idleTimeout = t
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
	cclog.ComponentDebug(r.name, "INIT", "listen on:", uri)

	http.HandleFunc(p, r.ServerHttp)

	r.server = &http.Server{
		Addr:        addr,
		Handler:     nil,
		IdleTimeout: r.config.idleTimeout,
	}
	r.server.SetKeepAlivesEnabled(r.config.KeepAlivesEnabled)

	return r, nil
}
