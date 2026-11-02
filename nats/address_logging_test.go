// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package nats

import (
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	natserver "github.com/nats-io/nats-server/v2/server"
)

type logCapture struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (capture *logCapture) Write(data []byte) (int, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.buffer.Write(data)
}

func (capture *logCapture) String() string {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.buffer.String()
}

func TestNatsAddressLogsHideCredentials(t *testing.T) {
	for _, key := range []string{"CC_NATS_USERNAME", "CC_NATS_PASSWORD", "CC_NATS_USERNAME_FILE", "CC_NATS_PASSWORD_FILE"} {
		t.Setenv(key, "")
	}

	for _, mode := range []string{"password", "token", "bare-password", "multiple-servers"} {
		t.Run(mode, func(t *testing.T) {
			const password = "synthetic-audit-password"
			const token = "synthetic-audit-token"
			options := &natserver.Options{Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true}
			if mode == "token" {
				options.Authorization = token
			} else {
				options.Username, options.Password = "audit", password
			}
			broker, err := natserver.NewServer(options)
			if err != nil {
				t.Fatal(err)
			}
			go broker.Start()
			if !broker.ReadyForConnections(time.Second) {
				t.Fatal("owned authentication broker failed to start")
			}
			t.Cleanup(broker.Shutdown)
			address, err := url.Parse(broker.ClientURL())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "token" {
				address.User = url.User(token)
			} else {
				address.User = url.UserPassword("audit", password)
			}
			configuredAddress := address.String()
			if mode == "bare-password" {
				configuredAddress = strings.TrimPrefix(configuredAddress, "nats://")
			} else if mode == "multiple-servers" {
				configuredAddress += "," + address.String()
			}
			var output logCapture
			previous := cclog.InfoLog.Writer()
			cclog.InfoLog.SetOutput(&output)
			t.Cleanup(func() { cclog.InfoLog.SetOutput(previous) })
			client, err := NewClient(&NatsConfig{Address: configuredAddress})
			if err != nil {
				// Do not emit the URL/error containing synthetic credential text.
				t.Fatal("valid controlled credential URL did not authenticate")
			}
			t.Cleanup(client.Close)
			if err := client.Connection().ForceReconnect(); err != nil {
				t.Fatal("owned NATS client could not trigger reconnect")
			}
			deadline := time.Now().Add(4 * time.Second)
			for !strings.Contains(output.String(), "NATS reconnected to") {
				if time.Now().After(deadline) {
					t.Fatal("owned broker did not complete reconnect and log its address")
				}
				time.Sleep(time.Millisecond)
			}
			captured := output.String()
			if !strings.Contains(captured, address.Host) {
				t.Error("redaction also removed the useful broker host and port")
			}
			if strings.Contains(captured, password) || strings.Contains(captured, token) {
				t.Error("startup or reconnect log contains URL credentials")
			}
		})
	}
}

func TestNatsConnectFailureLogsHideCredentials(t *testing.T) {
	for _, key := range []string{"CC_NATS_USERNAME", "CC_NATS_PASSWORD", "CC_NATS_USERNAME_FILE", "CC_NATS_PASSWORD_FILE"} {
		t.Setenv(key, "")
	}

	const isolatedFixture = "CC_AUDIT_NATS_LOG_FAILURE_FIXTURE"
	if os.Getenv(isolatedFixture) == "" {
		command := exec.Command(os.Args[0], "-test.run=^TestNatsConnectFailureLogsHideCredentials$", "-test.timeout=10s")
		command.Env = append(os.Environ(), isolatedFixture+"=1")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("isolated singleton log fixture: %v\n%s", err, output)
		}
		return
	}
	// The malformed percent escape is rejected before any dial or DNS lookup.
	const password = "synthetic-audit-password"
	Keys = NatsConfig{Address: "nats://audit:" + password + "@%zz"}
	var output logCapture
	previous := cclog.WarnLog.Writer()
	cclog.WarnLog.SetOutput(&output)
	t.Cleanup(func() { cclog.WarnLog.SetOutput(previous) })
	Connect()
	captured := output.String()
	if !strings.Contains(captured, "NATS connection failed") {
		t.Error("configured malformed address did not produce its expected failure log")
	}
	if strings.Contains(captured, password) {
		t.Error("connection failure log contains URL credentials")
	}
}
