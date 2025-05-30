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
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	influx "github.com/influxdata/line-protocol/v2/lineprotocol"
)

const HTTP_RECEIVER_PORT = "8080"

type HttpReceiverConfig struct {
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
}

type HttpReceiver struct {
	receiver
	// meta   map[string]string
	config HttpReceiverConfig
	server *http.Server
	wg     sync.WaitGroup
}

func (r *HttpReceiver) Init(name string, config json.RawMessage) error {
	r.name = fmt.Sprintf("HttpReceiver(%s)", name)

	// Set default values
	r.config.Port = HTTP_RECEIVER_PORT
	r.config.KeepAlivesEnabled = true
	// should be larger than the measurement interval to keep the connection open
	r.config.IdleTimeout = "120s"

	// Read config
	if len(config) > 0 {
		err := json.Unmarshal(config, &r.config)
		if err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return err
		}
	}
	if len(r.config.Port) == 0 {
		return errors.New("not all configuration variables set required by HttpReceiver")
	}

	// Check idle timeout config
	if len(r.config.IdleTimeout) > 0 {
		t, err := time.ParseDuration(r.config.IdleTimeout)
		if err == nil {
			cclog.ComponentDebug(r.name, "idleTimeout", t)
			r.config.idleTimeout = t
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
	// msgp, err := mp.NewMessageProcessor()
	// if err != nil {
	// 	return fmt.Errorf("initialization of message processor failed: %v", err.Error())
	// }
	// r.mp = msgp
	// if len(r.config.MessageProcessor) > 0 {
	// 	err = r.mp.FromConfigJSON(r.config.MessageProcessor)
	// 	if err != nil {
	// 		return fmt.Errorf("failed parsing JSON for message processor: %v", err.Error())
	// 	}
	// }
	// r.mp.AddAddMetaByCondition("true", "source", r.name)
	//
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

			m, err := r.mp.ProcessMessage(y)
			if err == nil && m != nil {
				r.sink <- m
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

func (r *HttpReceiver) Close() {
	r.server.Shutdown(context.Background())
}

func NewHttpReceiver(name string, config json.RawMessage) (Receiver, error) {
	r := new(HttpReceiver)
	err := r.Init(name, config)
	return r, err
}
