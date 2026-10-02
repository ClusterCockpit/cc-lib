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
	"github.com/nats-io/nkeys"
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
	// would return a client connecting in the background instead of an error.
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

func TestNewClient_RejectsInvalidNkeySeedFile(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "garbage.nk")
	if err := os.WriteFile(garbage, []byte("not a seed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"missing": filepath.Join(dir, "absent.nk"),
		"garbage": garbage,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewClient(&NatsConfig{Address: "nats://127.0.0.1:1", NkeySeedFile: path})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("expected the error to name %s, got %q", path, err.Error())
			}
		})
	}
}

func TestNewClient_UnreachableServerConnectsInBackground(t *testing.T) {
	kp, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}
	seedFile := filepath.Join(t.TempDir(), "user.nk")
	if err := os.WriteFile(seedFile, seed, 0o600); err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(&NatsConfig{Address: "nats://127.0.0.1:1", NkeySeedFile: seedFile})
	if err != nil {
		t.Fatalf("expected a client connecting in the background, got error: %v", err)
	}
	defer client.Close()

	if client.IsConnected() {
		t.Error("expected the client to be disconnected")
	}
}
