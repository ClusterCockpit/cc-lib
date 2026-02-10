# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cc-lib is a shared Go library (`github.com/ClusterCockpit/cc-lib/v2`) for the ClusterCockpit HPC monitoring ecosystem. It provides reusable packages for metric collection, data processing, storage, configuration, and system integration. It is a library only — there is no main binary.

Used by: cc-metric-collector, cc-metric-store, and ClusterCockpit web interface.

## Build & Test Commands

```bash
go build ./...                              # Build all packages
go test ./...                               # Run all tests
go test -v ./ccMessage                      # Test specific package
go test -v ./ccMessage -run TestJSONEncode   # Run single test
go test -cover ./...                        # Tests with coverage
```

No Makefile or special build tools. The `ccTopology` package requires hwloc C library (`sudo apt install hwloc`).

## Code Style Requirements

**Copyright header required on every Go file:**
```go
// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
```

**Import aliases used throughout the codebase:**
```go
cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
mp "github.com/ClusterCockpit/cc-lib/v2/messageProcessor"
```

**Formatting:** Code must be formatted with `gofumpt` (a stricter `gofmt`).

**Import grouping:** stdlib first, then third-party separated by blank line.

**Naming:** PascalCase exported, camelCase unexported. Avoid stuttering (`cache.New()` not `cache.NewCache()`).

**Testing:** Name tests `Test<FunctionName>_<Scenario>`. Use table-driven tests. Every package needs a package-level doc comment (linter enforced).

## Architecture

### Data Flow Pipeline

```
Receivers → CCMessage (chan) → MessageProcessor → Sinks
```

- **Receivers** collect metrics from sources (IPMI, Redfish, Prometheus, NATS, HTTP, EECPT)
- **CCMessage** is the internal message format extending InfluxDB line protocol with 5 types: Metric, Event, Log, Control, Query
- **MessageProcessor** transforms messages using expr-lang expressions (drop, rename, tag/meta manipulation)
- **Sinks** output to destinations (InfluxDB, NATS, Prometheus, HTTP, stdout, Ganglia)

Both receivers and sinks use a Manager pattern (`ReceiveManager`/`SinkManager`) for lifecycle and configuration.

### Key Package Dependencies

**Foundation layer** (used by most other packages):
- `schema` — Core types: Job, Cluster, SubCluster, MetricConfig, Float (NaN-aware JSON), User
- `ccMessage` — Message interface and concrete types
- `ccLogger` — Logging (thread-safe)
- `ccConfig` — JSON configuration with file references and hot-reload
- `ccUnits` — Unit prefix/measure system with conversion

**Processing layer:**
- `messageProcessor` — Expression-based message transformation pipeline (uses expr-lang)
- `resampler` — Time-series downsampling (SimpleResampler and LTTB algorithm)
- `lrucache` — Thread-safe LRU cache with TTL and HTTP middleware

**System integration:**
- `ccTopology` — Hardware topology via hwloc C bindings (cgo)
- `runtime` — .env loading, privilege dropping, systemd integration
- `nats` — Singleton NATS client wrapper
- `hostlist` — HPC hostlist expansion (e.g., `node[1-10]`)

### Notable Types

- `schema.Float` — Custom float64 that marshals NaN as JSON `null`; used extensively for metric data with missing values
- `schema.Job` — Central job representation with 20+ fields for HPC job metadata
- `schema.Cluster` / `schema.SubCluster` — Hardware configuration with topology and metric configs
- `ccMessage.CCMessage` — Interface with Name, Tags, Meta, Fields, Time, MessageType

### Thread Safety

- `ccLogger`, `lrucache`, `messageProcessor`, `nats.Client` are thread-safe
- `CCMessage` is NOT thread-safe — use `FromMessage()` to copy

### CI

Each package has its own GitHub Actions workflow in `.github/workflows/`. Tests run on ubuntu-latest with Go 1.23+.
