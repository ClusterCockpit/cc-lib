// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Config is the local fleet section of an application's configuration. It
// tells the client how to reach cc-backend; it never comes from the fleet
// service itself.
type Config struct {
	URL                    string `json:"url"`                                // cc-backend base URL
	Token                  string `json:"token,omitempty"`                    // JWT with the api role
	Cluster                string `json:"cluster,omitempty"`                  // empty selects infra scope
	Hostname               string `json:"hostname,omitempty"`                 // default os.Hostname()
	HeartbeatInterval      string `json:"heartbeat-interval,omitempty"`       // default 30s
	ConfigPollInterval     string `json:"config-poll-interval,omitempty"`     // default 60s
	HeartbeatSubject       string `json:"heartbeat-subject,omitempty"`        // empty: REST heartbeats only
	DiscoverySubjectPrefix string `json:"discovery-subject-prefix,omitempty"` // default DefaultDiscoveryPrefix
	CachePath              string `json:"cache-path,omitempty"`               // empty: no cache
	Timeout                string `json:"timeout,omitempty"`                  // HTTP timeout, default 10s
}

// EnvToken overrides Config.Token. CC_FLEET_TOKEN_FILE names a file holding
// the token instead; see util.SecretFromEnv for the precedence rules.
const EnvToken = "CC_FLEET_TOKEN"

// Defaults for the optional durations.
const (
	DefaultHeartbeatInterval  = 30 * time.Second
	DefaultConfigPollInterval = 60 * time.Second
	DefaultTimeout            = 10 * time.Second
)

const ConfigSchema = `{
    "type": "object",
    "description": "Connection to the cc-backend fleet service (registration, central configuration, heartbeat, discovery).",
    "properties": {
        "url": {
            "description": "Base URL of cc-backend (e.g. 'https://cc.example.org').",
            "type": "string"
        },
        "token": {
            "description": "JWT with the 'api' role. Overridden by the CC_FLEET_TOKEN environment variable when set, or by the contents of the file named by CC_FLEET_TOKEN_FILE.",
            "type": "string"
        },
        "cluster": {
            "description": "Cluster this service belongs to. Empty registers the service in infra scope.",
            "type": "string"
        },
        "hostname": {
            "description": "Hostname to register with. Defaults to the system hostname.",
            "type": "string"
        },
        "heartbeat-interval": {
            "description": "Interval between heartbeats; keep it well below cc-backend's stale-after (default '30s').",
            "type": "string"
        },
        "config-poll-interval": {
            "description": "Interval between configuration polls (default '60s').",
            "type": "string"
        },
        "heartbeat-subject": {
            "description": "NATS subject for heartbeats (e.g. 'cc.fleet.event'). Empty sends heartbeats over REST.",
            "type": "string"
        },
        "discovery-subject-prefix": {
            "description": "NATS subject prefix of discovery rosters (default 'cc.fleet.discovery').",
            "type": "string"
        },
        "cache-path": {
            "description": "File caching the last received configuration, so the service can start while cc-backend is unreachable. Empty disables the cache.",
            "type": "string"
        },
        "timeout": {
            "description": "Timeout of a single HTTP request to cc-backend (default '10s').",
            "type": "string"
        }
    },
    "required": ["url"]
}`

// ParseConfig decodes and validates a fleet configuration section. Unknown
// keys and malformed durations are errors.
func ParseConfig(raw json.RawMessage) (Config, error) {
	var cfg Config
	if len(raw) == 0 {
		return cfg, errors.New("fleet: empty configuration")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("fleet: decoding configuration: %w", err)
	}
	if cfg.URL == "" {
		return cfg, errors.New("fleet: configuration is missing 'url'")
	}
	if _, err := cfg.durations(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type durations struct {
	heartbeat, poll, timeout time.Duration
}

func (c Config) durations() (durations, error) {
	var d durations
	var err error
	if d.heartbeat, err = parseDuration("heartbeat-interval", c.HeartbeatInterval, DefaultHeartbeatInterval); err != nil {
		return d, err
	}
	if d.poll, err = parseDuration("config-poll-interval", c.ConfigPollInterval, DefaultConfigPollInterval); err != nil {
		return d, err
	}
	if d.timeout, err = parseDuration("timeout", c.Timeout, DefaultTimeout); err != nil {
		return d, err
	}
	return d, nil
}

func parseDuration(key, value string, def time.Duration) (time.Duration, error) {
	if value == "" {
		return def, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("fleet: invalid %s %q: %w", key, value, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("fleet: %s must be positive, got %q", key, value)
	}
	return d, nil
}
