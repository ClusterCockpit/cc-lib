// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)
package sinks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusSinkConfig struct {
	defaultSinkConfig
	Host             string `json:"host,omitempty"`
	Port             string `json:"port"`
	Path             string `json:"path,omitempty"`
	GroupAsNameSpace bool   `json:"group_as_namespace,omitempty"`
	// User       string `json:"user,omitempty"`
	// Password   string `json:"password,omitempty"`
	// FlushDelay string `json:"flush_delay,omitempty"`
}

type PrometheusSink struct {
	sink
	config       PrometheusSinkConfig
	labelMetrics map[string]*prometheus.GaugeVec
	nodeMetrics  map[string]prometheus.Gauge
	promWg       sync.WaitGroup
	promServer   *http.Server
}

func intToFloat64(input interface{}) (float64, error) {
	switch value := input.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	}
	return 0, errors.New("cannot cast value to float64")
}

func getLabelValue(metric lp.CCMessage) []string {
	labelValues := []string{}
	if tid, tidok := metric.GetTag("type-id"); tidok && metric.HasTag("type") {
		labelValues = append(labelValues, tid)
	}
	if d, ok := metric.GetTag("device"); ok {
		labelValues = append(labelValues, d)
	} else if d, ok := metric.GetMeta("device"); ok {
		labelValues = append(labelValues, d)
	}
	return labelValues
}

func getLabelNames(metric lp.CCMessage) []string {
	labelNames := []string{}
	if t, tok := metric.GetTag("type"); tok && metric.HasTag("type-id") {
		labelNames = append(labelNames, t)
	}
	if _, ok := metric.GetTag("device"); ok {
		labelNames = append(labelNames, "device")
	} else if _, ok := metric.GetMeta("device"); ok {
		labelNames = append(labelNames, "device")
	}
	return labelNames
}

func (s *PrometheusSink) newMetric(metric lp.CCMessage) error {
	var value float64 = 0
	name := metric.Name()
	opts := prometheus.GaugeOpts{
		Name: name,
	}
	labels := getLabelNames(metric)
	labelValues := getLabelValue(metric)
	if len(labels) > 0 && len(labels) != len(labelValues) {
		return fmt.Errorf("cannot detect metric labels for metric %s", name)
	}

	if metricValue, ok := metric.GetField("value"); ok {
		if floatValue, err := intToFloat64(metricValue); err == nil {
			value = floatValue
		} else {
			return fmt.Errorf("metric %s with value '%v' cannot be casted to float64", name, metricValue)
		}
	}
	if s.config.GroupAsNameSpace && metric.HasMeta("group") {
		g, _ := metric.GetMeta("group")
		opts.Namespace = strings.ToLower(g)
	}

	if len(labels) > 0 {
		new := prometheus.NewGaugeVec(opts, labels)
		new.WithLabelValues(labelValues...).Set(value)
		s.labelMetrics[name] = new
		prometheus.Register(new)
	} else {
		new := prometheus.NewGauge(opts)
		new.Set(value)
		s.nodeMetrics[name] = new
		prometheus.Register(new)
	}
	return nil
}

func (s *PrometheusSink) updateMetric(metric lp.CCMessage) error {
	value := 0.0
	name := metric.Name()
	labelValues := getLabelValue(metric)

	if metricValue, ok := metric.GetField("value"); ok {
		if floatValue, err := intToFloat64(metricValue); err == nil {
			value = floatValue
		} else {
			return fmt.Errorf("metric %s with value '%v' cannot be casted to float64", name, metricValue)
		}
	}

	if len(labelValues) > 0 {
		if _, ok := s.labelMetrics[name]; !ok {
			err := s.newMetric(metric)
			if err != nil {
				return err
			}
		}
		s.labelMetrics[name].WithLabelValues(labelValues...).Set(value)
	} else {
		if _, ok := s.labelMetrics[name]; !ok {
			err := s.newMetric(metric)
			if err != nil {
				return err
			}
		}
		s.nodeMetrics[name].Set(value)
	}
	return nil
}

func (s *PrometheusSink) Write(m lp.CCMessage) error {
	msg, err := s.mp.ProcessMessage(m)
	if err == nil && msg != nil {
		err = s.updateMetric(m)
	}
	return err
}

func (s *PrometheusSink) Flush() error {
	return nil
}

func (s *PrometheusSink) Close() {
	cclog.ComponentDebug(s.name, "CLOSE")
	s.promServer.Shutdown(context.Background())
	s.promWg.Wait()
}

func NewPrometheusSink(name string, config json.RawMessage) (Sink, error) {
	s := new(PrometheusSink)
	s.name = "PrometheusSink"
	if len(config) > 0 {
		d := json.NewDecoder(bytes.NewReader(config))
		d.DisallowUnknownFields()
		if err := d.Decode(&s.config); err != nil {
			cclog.ComponentError(s.name, "Error reading config:", err.Error())
			return nil, err
		}
	}
	if len(s.config.Port) == 0 {
		err := errors.New("not all configuration variables set required by PrometheusSink")
		cclog.ComponentError(s.name, err.Error())
		return nil, err
	}
	p, err := mp.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("initialization of message processor failed: %v", err.Error())
	}
	s.mp = p
	if len(s.config.MessageProcessor) > 0 {
		err = p.FromConfigJSON(s.config.MessageProcessor)
		if err != nil {
			return nil, fmt.Errorf("failed parsing JSON for message processor: %v", err.Error())
		}
	}
	for _, k := range s.config.MetaAsTags {
		s.mp.AddMoveMetaToTags("true", k, k)
	}
	s.labelMetrics = make(map[string]*prometheus.GaugeVec)
	s.nodeMetrics = make(map[string]prometheus.Gauge)
	s.promWg.Add(1)
	go func() {
		router := mux.NewRouter()
		// Prometheus endpoint
		router.Path("/" + s.config.Path).Handler(promhttp.Handler())

		url := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
		cclog.ComponentDebug(s.name, "Serving Prometheus metrics at", fmt.Sprintf("%s:%s/%s", s.config.Host, s.config.Port, s.config.Path))
		s.promServer = &http.Server{Addr: url, Handler: router}
		err := s.promServer.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			cclog.ComponentError(s.name, err.Error())
		}
		s.promWg.Done()
	}()
	return s, nil
}
