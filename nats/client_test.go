// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package nats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/util"
)

func TestResolveCredentials_Precedence(t *testing.T) {
	tests := []struct {
		name         string
		cfg          NatsConfig
		envUser      string
		envPassFile  string
		wantUser     string
		wantPassword string
	}{
		{
			name:         "config only",
			cfg:          NatsConfig{Username: "cfg-user", Password: "cfg-pass"},
			wantUser:     "cfg-user",
			wantPassword: "cfg-pass",
		},
		{
			name:         "environment overrides config",
			cfg:          NatsConfig{Username: "cfg-user", Password: "cfg-pass"},
			envUser:      "env-user",
			wantUser:     "env-user",
			wantPassword: "cfg-pass",
		},
		{
			name:         "secret file overrides config",
			cfg:          NatsConfig{Username: "cfg-user", Password: "cfg-pass"},
			envPassFile:  "file-pass\n",
			wantUser:     "cfg-user",
			wantPassword: "file-pass",
		},
		{
			name:         "nothing configured",
			wantUser:     "",
			wantPassword: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envUser != "" {
				t.Setenv(EnvUsername, tt.envUser)
			}
			if tt.envPassFile != "" {
				path := filepath.Join(t.TempDir(), "password")
				if err := os.WriteFile(path, []byte(tt.envPassFile), 0o600); err != nil {
					t.Fatalf("writing secret file: %v", err)
				}
				t.Setenv(EnvPassword+util.EnvFileSuffix, path)
			}

			cfg := tt.cfg
			user, password, err := resolveCredentials(&cfg)
			if err != nil {
				t.Fatalf("resolveCredentials failed: %v", err)
			}
			if user != tt.wantUser {
				t.Errorf("expected username %q, got %q", tt.wantUser, user)
			}
			if password != tt.wantPassword {
				t.Errorf("expected password %q, got %q", tt.wantPassword, password)
			}

			// The resolved secret must never be written back into the config.
			if cfg.Username != tt.cfg.Username || cfg.Password != tt.cfg.Password {
				t.Error("expected the config to be left unmodified")
			}
		})
	}
}

func TestResolveCredentials_UnreadableSecretFile(t *testing.T) {
	t.Setenv(EnvPassword+util.EnvFileSuffix, filepath.Join(t.TempDir(), "absent"))

	cfg := NatsConfig{Username: "cfg-user", Password: "cfg-pass"}
	_, _, err := resolveCredentials(&cfg)
	if err == nil {
		t.Fatal("expected an error for an unreadable secret file, got nil")
	}
	// The variable must be named so an operator can find the misconfiguration,
	// and the config value must not be used as a silent fallback.
	if !strings.Contains(err.Error(), EnvPassword) {
		t.Errorf("expected the error to name %s, got %q", EnvPassword, err.Error())
	}
}

func TestNewClient_RejectsUnreadableSecretFileBeforeConnecting(t *testing.T) {
	t.Setenv(EnvPassword+util.EnvFileSuffix, filepath.Join(t.TempDir(), "absent"))

	// An unroutable address: if credential resolution did not fail first, this
	// would block on a connection attempt instead of returning promptly.
	_, err := NewClient(&NatsConfig{Address: "nats://127.0.0.1:1", Password: "cfg-pass"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), EnvPassword) {
		t.Errorf("expected the error to name %s, got %q", EnvPassword, err.Error())
	}
}

func TestNewClient_RequiresAddress(t *testing.T) {
	if _, err := NewClient(&NatsConfig{}); err == nil {
		t.Error("expected an error for an empty address, got nil")
	}
}
