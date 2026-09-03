// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package nats

import (
	"bytes"
	"encoding/json"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// NatsConfig holds the configuration for connecting to a NATS server.
type NatsConfig struct {
	Address       string `json:"address"`         // NATS server address (e.g., "nats://localhost:4222")
	Username      string `json:"username"`        // Username for authentication (optional)
	Password      string `json:"password"`        // Password for authentication (optional)
	CredsFilePath string `json:"creds-file-path"` // Path to credentials file (optional)
}

// Keys holds the global NATS configuration loaded via Init.
var Keys NatsConfig

// Environment variables that override the corresponding configuration values.
// Each also accepts a "_FILE" variant naming a file that holds the value, so a
// credential can be supplied from a Docker or Kubernetes secret mount or from
// systemd LoadCredential; see util.SecretFromEnv for the precedence rules.
//
// The names are prefixed CC_ because cc-lib is linked into several
// applications, whose environments it must not silently claim names in.
const (
	EnvUsername = "CC_NATS_USERNAME"
	EnvPassword = "CC_NATS_PASSWORD"
)

const ConfigSchema = `{
    "type": "object",
    "description": "Configuration for NATS messaging client.",
    "properties": {
        "address": {
            "description": "Address of the NATS server (e.g., 'nats://localhost:4222').",
            "type": "string"
        },
        "username": {
            "description": "Username for NATS authentication (optional). Overridden by the CC_NATS_USERNAME environment variable when set, or by the contents of the file named by CC_NATS_USERNAME_FILE.",
            "type": "string"
        },
        "password": {
            "description": "Password for NATS authentication (optional). Overridden by the CC_NATS_PASSWORD environment variable when set, or by the contents of the file named by CC_NATS_PASSWORD_FILE.",
            "type": "string"
        },
        "creds-file-path": {
            "description": "Path to NATS credentials file for authentication (optional).",
            "type": "string"
        }
    },
    "required": ["address"]
}`

// Init initializes the global Keys configuration from JSON.
func Init(rawConfig json.RawMessage) error {
	var err error

	if rawConfig != nil {
		dec := json.NewDecoder(bytes.NewReader(rawConfig))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&Keys); err != nil {
			cclog.Errorf("Error while initializing nats client: %s", err.Error())
		}
	}

	return err
}
