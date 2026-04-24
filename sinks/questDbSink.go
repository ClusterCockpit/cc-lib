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
	"time"

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

	// Additional config options, for QuestDBSink:

	// Address to connect to. Should be in the "host:port" format
	Address string `json:"address,omitempty"`
	// Authentication options for QuestDB:
	// Basic authentication with username and password
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Authentication with bearer token in HTTP header
	BearerToken string `json:"bearer_token,omitempty"`
	// Auto flush configuration
	// Interval at which the sender automatically flushes its buffer
	AutoFlushInterval string `json:"auto_flush_interval,omitempty"`
	// Number of rows after which the sender automatically flushes its buffer
	AutoFlushRows int `json:"auto_flush_rows,omitempty"`
	// Enable TLS for secure connections
	UseTLS bool `json:"use_tls,omitempty"`
}

type QuestDBSink struct {
	// declares elements 'name' and 'meta_as_tags' (string to bool map!)
	// See: metricSink.go
	sink
	config QuestDBSinkConfig
	sender qdb.LineSender
	ctx    context.Context
}

// Column name cannot contain any of the following characters:
// '\n', '\r', '?', '.', ',', ”', '"', '\', '/', ':', ')', '(', '+',
// '-', '*' '%%', '~', or a non-printable char.
var sanitizeKey *strings.Replacer = strings.NewReplacer(
	"-id", "ID",
	"-", "_",
)

// Code to submit a single CCMetric to the sink
func (s *QuestDBSink) Write(point lp.CCMessage) error {
	// Submit the point to the message processor to apply rules
	msg, err := s.mp.ProcessMessage(point)
	if err == nil && msg != nil {

		// Metric name is used as table name in QuestDB
		s.sender.Table(msg.Name())

		// Add tags as symbol columns
		for k, v := range msg.Tags() {
			s.sender.Symbol(sanitizeKey.Replace(k), v)
		}

		// Add fields as value columns
		for k, v := range msg.Fields() {
			k = sanitizeKey.Replace(k)
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
		if err := s.sender.At(s.ctx, msg.Time()); err != nil {
			return fmt.Errorf("failed to write point: %w", err)
		}
	}
	return nil
}

// If the sink uses batched sends internally, you can tell to flush its buffers
func (s *QuestDBSink) Flush() error {
	return s.sender.Flush(s.ctx)
}

// Close sink: close network connection, close files, close libraries, ...
func (s *QuestDBSink) Close() {
	if err := s.Flush(); err != nil {
		cclog.ComponentError(s.name, fmt.Errorf("flush failed with error: %v", err))
	}
	if err := s.sender.Close(s.ctx); err != nil {
		cclog.ComponentError(s.name, fmt.Errorf("close failed with error: %v", err))
	}
	cclog.ComponentDebug(s.name, "CLOSE")
}

// NewQuestDBSink initializes the QuestDB sink with the given name and configuration.
// It returns an error if the configuration is invalid or if the connection to the QuestDB server cannot be established.
func NewQuestDBSink(name string, config json.RawMessage) (Sink, error) {
	s := new(QuestDBSink)

	// Set name of QuestDBSink
	s.name = fmt.Sprintf("QuestDBSink(%s)", name) // Always specify a name here

	// Set defaults in s.config
	s.config.Address = "localhost:9000"
	s.config.AutoFlushInterval = "5s"

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
	for _, k := range s.config.MetaAsTags {
		s.mp.AddMoveMetaToTags("true", k, k)
	}

	// Configure connection options
	options := []qdb.LineSenderOption{
		qdb.WithAddress(s.config.Address),
	}
	if s.config.UseTLS {
		options = append(options, qdb.WithTls())
	} else {
		options = append(options, qdb.WithHttp())
	}
	autoFlushInterval, err := time.ParseDuration(s.config.AutoFlushInterval)
	if err != nil {
		return nil, fmt.Errorf("failed parsing auto flush interval: %w", err)
	}
	options = append(options, qdb.WithAutoFlushInterval(autoFlushInterval))
	if s.config.AutoFlushRows > 0 {
		options = append(options, qdb.WithAutoFlushRows(s.config.AutoFlushRows))
	}
	if (s.config.Username != "" && s.config.Password == "") ||
		(s.config.Username == "" && s.config.Password != "") {
		return nil, fmt.Errorf("incomplete basic authentication credentials")
	}
	if s.config.Username != "" && s.config.Password != "" && s.config.BearerToken != "" {
		return nil, fmt.Errorf("conflicting authentication methods: both basic auth and bearer token provided")
	}
	if s.config.Username != "" && s.config.Password != "" {
		options = append(options,
			qdb.WithBasicAuth(s.config.Username, s.config.Password))
	}
	if s.config.BearerToken != "" {
		options = append(options,
			qdb.WithBearerToken(s.config.BearerToken))
	}

	s.ctx = context.Background()

	// Connect to QuestDB server
	sender, err := qdb.NewLineSender(s.ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed creating new line sender: %w", err)
	}
	s.sender = sender

	return s, nil
}
