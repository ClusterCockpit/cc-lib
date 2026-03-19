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
	"fmt"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	mp "github.com/ClusterCockpit/cc-lib/v2/messageProcessor"

	// See https://pkg.go.dev/github.com/questdb/go-questdb-client/v4
	qdb "github.com/questdb/go-questdb-client/v4"
)

type QuestDBSinkConfig struct {
	// defines JSON tags for 'type' and 'meta_as_tags' (string list)
	// See: metricSink.go
	defaultSinkConfig
	// Additional config options, for QuestDBSink
	Address     string `json:"address,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	BearerToken string `json:"bearer_token,omitempty"`
}

type QuestDBSink struct {
	// declares elements 'name' and 'meta_as_tags' (string to bool map!)
	// See: metricSink.go
	sink
	config QuestDBSinkConfig
	sender qdb.LineSender
}

// Code to submit a single CCMetric to the sink
func (s *QuestDBSink) Write(point lp.CCMessage) error {
	// based on s.meta_as_tags use meta infos as tags
	// moreover, submit the point to the message processor
	// to apply drop/modify rules
	msg, err := s.mp.ProcessMessage(point)
	if err == nil && msg != nil {
		s.sender.Table(msg.Name())
		for k, v := range msg.Tags() {
			s.sender.Symbol(strings.ReplaceAll(k, "-id", "ID"), v)
		}
		for k, v := range msg.Fields() {
			switch v := v.(type) {
			case float64:
				s.sender.Float64Column(k, v)
			case uint64:
				s.sender.Int64Column(k, int64(v))
			case int64:
				s.sender.Int64Column(k, v)
			case string:
				s.sender.StringColumn(k, v)
			default:
				cclog.ComponentError(s.name, fmt.Sprintf("Unsupported data type %T", v))
			}
		}
		if err := s.sender.At(context.TODO(), msg.Time()); err != nil {
			cclog.ComponentError(s.name, fmt.Sprintf("write failed: %v", err))
		}
		s.sender.Flush(context.TODO())
	}
	return nil
}

// If the sink uses batched sends internally, you can tell to flush its buffers
func (s *QuestDBSink) Flush() error {
	return s.sender.Flush(context.TODO())
}

// Close sink: close network connection, close files, close libraries, ...
func (s *QuestDBSink) Close() {
	if err := s.Flush(); err != nil {
		cclog.ComponentError(s.name, fmt.Errorf("flush failed with error: %v", err))
	}
	if err := s.sender.Close(context.TODO()); err != nil {
		cclog.ComponentError(s.name, fmt.Errorf("close failed with error: %v", err))
	}
	cclog.ComponentDebug(s.name, "CLOSE")
}

// New function to create a new instance of the sink
// Initialize the sink by giving it a name and reading in the config JSON
func NewQuestDBSink(name string, config json.RawMessage) (Sink, error) {
	s := new(QuestDBSink)

	// Set name of QuestDBSink
	// The name should be chosen in such a way that different instances of QuestDBSink can be distinguished
	s.name = fmt.Sprintf("QuestDBSink(%s)", name) // Always specify a name here

	// Set defaults in s.config
	s.config.Address = "localhost:9000"

	// Read in the config JSON
	if len(config) > 0 {
		d := json.NewDecoder(bytes.NewReader(config))
		d.DisallowUnknownFields()
		if err := d.Decode(&s.config); err != nil {
			cclog.ComponentError(s.name, "Error reading config:", err.Error())
			return nil, err
		}
	}

	// Initialize and configure the message processor
	p, err := mp.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("initialization of message processor failed: %v", err.Error())
	}
	s.mp = p

	// Add message processor configuration
	if len(s.config.MessageProcessor) > 0 {
		err = p.FromConfigJSON(s.config.MessageProcessor)
		if err != nil {
			return nil, fmt.Errorf("failed parsing JSON for message processor: %w", err)
		}
	}
	// Add rules to move meta information to tag space
	// Replacing the legacy 'meta_as_tags' configuration
	for _, k := range s.config.MetaAsTags {
		s.mp.AddMoveMetaToTags("true", k, k)
	}

	// Establish connection to the QuestDB server
	options := []qdb.LineSenderOption{
		qdb.WithHttp(),
		qdb.WithAddress(s.config.Address),
	}
	if s.config.Username != "" && s.config.Password != "" {
		options = append(options,
			qdb.WithBasicAuth(s.config.Username, s.config.Password))
	}
	if s.config.BearerToken != "" {
		options = append(options,
			qdb.WithBearerToken(s.config.BearerToken))
	}
	sender, err := qdb.NewLineSender(
		context.TODO(),
		options...)
	if err != nil {
		return s, fmt.Errorf("failed creating new line sender: %w", err)
	}
	s.sender = sender

	// Return (nil, meaningful error message) in case of errors
	return s, nil
}
