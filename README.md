<!--
title: cc-lib
description: Component library for various ClusterCockpit applications
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 1
hugo_path: docs/reference/cc-lib/_index.md
-->

[![GoDoc](https://godoc.org/github.com/ClusterCockpit/cc-lib?status.svg)](https://godoc.org/github.com/ClusterCockpit/cc-lib)
[![Go Report Card](https://goreportcard.com/badge/github.com/ClusterCockpit/cc-lib)](https://goreportcard.com/report/github.com/ClusterCockpit/cc-lib)

# cc-lib

Common ClusterCockpit golang packages providing reusable components for building HPC monitoring and metric collection applications.

## Overview

cc-lib is a collection of Go packages developed for the [ClusterCockpit](https://github.com/ClusterCockpit) project. These packages provide essential functionality for:

- **Metric Collection**: Receivers for various protocols (IPMI, Redfish, Prometheus, etc.)
- **Data Processing**: Message processing pipelines, resampling, and transformations
- **Data Storage**: Sinks for InfluxDB, NATS, Prometheus, and more
- **Configuration**: Flexible configuration management with validation
- **Utilities**: Caching, logging, topology detection, and helper functions

The library is designed to be modular, allowing you to use individual packages as needed in your own projects.

## Packages

### Core Messaging & Processing

| Package                                | Description                                                                 |
| -------------------------------------- | --------------------------------------------------------------------------- |
| [ccMessage](./ccMessage)               | Message types and protocols for metrics, logs, events, and control messages |
| [messageProcessor](./messageProcessor) | Expression-based message processing and transformation pipeline             |
| [schema](./schema)                     | JSON schema definitions and validation for ClusterCockpit data structures   |

### Metric Collection

| Package                    | Description                                                         |
| -------------------------- | ------------------------------------------------------------------- |
| [receivers](./receivers)   | Metric receivers for IPMI, Redfish, Prometheus, and other protocols |
| [ccTopology](./ccTopology) | System topology detection and hardware information gathering        |

### Data Storage & Output

| Package                  | Description                                                        |
| ------------------------ | ------------------------------------------------------------------ |
| [sinks](./sinks)         | Metric sinks for InfluxDB, NATS, Prometheus, HTTP, and file output |
| [resampler](./resampler) | Data resampling and aggregation utilities                          |

### Configuration & Logging

| Package                | Description                                              |
| ---------------------- | -------------------------------------------------------- |
| [ccConfig](./ccConfig) | Configuration file management with hot-reloading support |
| [ccLogger](./ccLogger) | Structured logging with multiple output levels           |

### Utilities

| Package                | Description                                                             |
| ---------------------- | ----------------------------------------------------------------------- |
| [lrucache](./lrucache) | Thread-safe LRU cache with TTL support and HTTP middleware              |
| [hostlist](./hostlist) | Hostlist expansion for compact host specifications (e.g., `node[1-10]`) |
| [ccUnits](./ccUnits)   | Unit conversion and handling for metrics                                |
| [util](./util)         | Common utility functions and helpers                                    |
| [runtime](./runtime)   | Runtime environment setup, privilege dropping, and systemd integration  |

## Installation

```bash
go get github.com/ClusterCockpit/cc-lib
```

**Requirements:**

- Go 1.24.0 or higher

## Quick Start

### Using the LRU Cache

```go
import "github.com/ClusterCockpit/cc-lib/lrucache"

cache := lrucache.New(1000) // maxmemory in arbitrary units

value := cache.Get("key", func() (interface{}, time.Duration, int) {
    // Compute expensive value
    result := fetchFromDatabase()
    return result, 10 * time.Minute, len(result)
})
```

### Expanding Hostlists

```go
import "github.com/ClusterCockpit/cc-lib/hostlist"

hosts, err := hostlist.Expand("node[1-10],gpu[1-4]")
// Returns: [gpu1, gpu2, gpu3, gpu4, node1, node2, ..., node10]
```

### Creating Messages

```go
import "github.com/ClusterCockpit/cc-lib/ccMessage"

msg, err := ccMessage.NewMessage(
    "temperature",
    map[string]string{"hostname": "node01", "type": "node"},
    map[string]string{"unit": "degC"},
    map[string]interface{}{"value": 45.2},
    time.Now(),
)
```

### Using Configuration Management

```go
import "github.com/ClusterCockpit/cc-lib/ccConfig"

config := ccConfig.New()
config.AddFile("config.json")

// Access configuration
value := config.Get("key")

// Watch for changes
config.Watch(func() {
    log.Println("Configuration changed")
})
```

## Documentation

- **API Documentation**: [pkg.go.dev/github.com/ClusterCockpit/cc-lib](https://pkg.go.dev/github.com/ClusterCockpit/cc-lib)
- **Package READMEs**: Each package has its own README with detailed documentation and examples

### Package Documentation

- [ccConfig](./ccConfig/README.md) - Configuration management
- [ccLogger](./ccLogger/README.md) - Logging utilities
- [ccMessage](./ccMessage/README.md) - Message types and protocols
- [ccTopology](./ccTopology/README.md) - System topology detection
- [ccUnits](./ccUnits/README.md) - Unit conversion
- [hostlist](./hostlist/README.md) - Hostlist expansion
- [lrucache](./lrucache/README.md) - LRU cache with TTL
- [messageProcessor](./messageProcessor/README.md) - Message processing
- [receivers](./receivers/README.md) - Metric receivers
- [runtime](./runtime/README.md) - Runtime environment setup
- [schema](./schema/README.md) - JSON schema validation
- [sinks](./sinks/README.md) - Metric sinks
- [util](./util/README.md) - Utility functions

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Run tests for a specific package:

```bash
go test -v ./lrucache
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

### Development

1. Clone the repository
2. Make your changes
3. Run tests: `go test ./...`
4. Submit a pull request

## Projects Using cc-lib

- [cc-metric-collector](https://github.com/ClusterCockpit/cc-metric-collector) - Metric collection daemon
- [cc-metric-store](https://github.com/ClusterCockpit/cc-metric-store) - Metric storage backend
- [ClusterCockpit](https://github.com/ClusterCockpit/ClusterCockpit) - Web interface and monitoring system

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (C) NHR@FAU, University Erlangen-Nuremberg.

## Acknowledgments

Developed by the [National High Performance Computing (NHR) center at FAU](https://hpc.fau.de/).

Additional contributors:

- Holger Obermaier (NHR@KIT)
