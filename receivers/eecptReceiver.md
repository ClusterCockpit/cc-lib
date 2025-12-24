<!--
---
title: Message receiver for messages from EECPT instrumentation library
description: Receiving messages from EECPT instrumentation library
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/eecpt.md
---
-->

## `eecpt` receiver

The `eecpt` (Energy-Efficient Computing Phase Transition) receiver is a specialized HTTP receiver designed to detect phase transitions in application behavior using statistical analysis. It listens for metrics from the EECPT instrumentation library and performs real-time chi-square tests.

### Architecture

The receiver buffers incoming metrics and periodically runs an analysis. If the statistical test exceeds a pre-calculated threshold (at 95% confidence level), it generates a "phase transition" event message.

### Configuration Structure

```json
{
  "my_eecpt_receiver": {
    "type": "eecpt",
    "address" : "0.0.0.0",
    "port" : "8080",
    "path" : "/write",
    "idle_timeout": "120s",
    "keep_alives_enabled": true,
    "username": "myUser",
    "password": "myPW",
    "analysis_buffer_size": 10,
    "analysis_interval": "5m",
    "analysis_metric": "region_metric",
    "process_messages": []
  }
}
```

### Configuration Options

- `type`: Must be `eecpt`.
- `address`: IP address to listen on (default: empty for all interfaces).
- `port`: Port to listen on (default: `8080`).
- `path`: URL path for the write endpoint (default: `/write`).
- `idle_timeout`: Maximum idle time for keep-alive connections (default: `120s`).
- `keep_alives_enabled`: Whether to enable HTTP keep-alives (default: `true`).
- `username`: Optional username for basic authentication.
- `password`: Optional password for basic authentication.
- `analysis_buffer_size`: Number of metric values to keep in the history buffer for each task (default: `4`, minimum: `4`).
- `analysis_interval`: How often to perform the phase transition analysis (default: `5m`).
- `analysis_metric`: The name of the metric to perform analysis on (default: `region_metric`).
- `process_messages`: Optional message processing rules.

### Data Format

The EECPT library typically sends data in InfluxDB line protocol. The receiver looks for `jobid` (or `application`) and `rank` (or `pid`) to identify tasks within a job.

Example incoming metric:
```
region_metric,hostname=node01,jobid=1234,rank=0 value=0.45 1694777161164284635
```

### Generated Events

When a phase transition is detected, the receiver generates an event:
- **Name**: `region`
- **Tags**: `type=node`, `stype=application`, `stype-id=<jobid>`
- **Fields**: `value="region changed"`
- **Timestamp**: Current time
