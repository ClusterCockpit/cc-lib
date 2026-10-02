// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet_test

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	ccconfig "github.com/ClusterCockpit/cc-lib/v2/ccConfig"
	"github.com/ClusterCockpit/cc-lib/v2/fleet"
	"github.com/ClusterCockpit/cc-lib/v2/nats"
)

// natsClient is set once the application has connected to NATS, which may
// happen only after the fleet configuration delivered the NATS settings.
var natsClient atomic.Pointer[nats.Client]

func applyConfig(json.RawMessage)             {}
func useProviders(providers []fleet.Provider) {}

// Example shows the lifecycle of a fleet member: bootstrap, run, apply
// updates from the main loop, close on shutdown.
func Example() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := fleet.ParseConfig(ccconfig.GetPackageConfig("fleet"))
	if err != nil {
		log.Fatal(err)
	}
	client, err := fleet.New(fleet.Options{
		Config:      cfg,
		ServiceType: fleet.ServiceMetricCollector,
		Meta:        map[string]string{"version": "1.0.0"},
		NATS:        natsClient.Load, // nil until NATS is connected: REST heartbeats
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start with the fleet configuration, the cached one, or none at all.
	initial, err := client.Bootstrap(ctx, 10*time.Second)
	if errors.Is(err, fleet.ErrUnreachable) {
		log.Printf("cc-backend unreachable, starting with %s configuration", initial.Source)
	} else if err != nil {
		log.Printf("fleet bootstrap: %v", err)
	}
	applyConfig(initial.Config) // nil: no fleet configuration, use local defaults

	go func() { _ = client.Run(ctx) }()
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.Close(closeCtx); err != nil {
			log.Printf("fleet deregistration: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case u := <-client.Configs():
			applyConfig(u.Config)
		case providers := <-client.Rosters():
			useProviders(providers) // full replacement, not a delta
		}
	}
}
