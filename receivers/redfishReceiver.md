<!--
---
title: Message receiver for Redfish endpoints
description: Query metrics from remote Redfish sources
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/redfish.md
---
-->

## `redfish` receiver

The `redfish` receiver uses the [Redfish specification](https://www.dmtf.org/standards/redfish) to query thermal and power metrics from modern hardware management interfaces. It polls multiple devices in parallel to maintain high throughput.

### Configuration Structure

```json
{
  "my_redfish_receiver": {
    "type": "redfish",
    "interval": "30s",
    "fanout": 64,
    "http_insecure": true,
    "http_timeout": "10s",
    "username": "admin",
    "password": "password",
    "endpoint": "https://%h-bmc",
    "exclude_metrics": [ "min_consumed_watts" ],
    "process_messages": [],
    "client_config": [
      {
        "host_list": "node[01-04]"
      },
      {
        "host_list": "node05",
        "disable_power_metrics": true
      },
      {
        "host_list": "node06",
        "username": "user2",
        "password": "password2",
        "disable_thermal_metrics": true
      }
    ]
  }
}
```

### Global Configuration Options

- `type`: Must be `redfish`.
- `interval`: Polling interval (default: `30s`).
- `fanout`: Maximum number of simultaneous connections (default: `64`).
- `http_insecure`: Skip SSL certificate verification (default: `true`).
- `http_timeout`: Timeout for HTTP requests (default: `10s`).
- `process_messages`: Optional message processing rules.

### Global and Per-Device Options

These settings can be defined globally and overridden in `client_config`:

- `endpoint`: URL template for the Redfish service. `%h` is replaced by the hostname.
- `username`: Username for authentication.
- `password`: Password for authentication.
- `disable_power_metrics`: Disable collection of power metrics.
- `disable_processor_metrics`: Disable collection of processor metrics.
- `disable_thermal_metrics`: Disable collection of thermal metrics.
- `exclude_metrics`: List of specific metrics to exclude.

### Per-Device Options (`client_config`)

- `host_list`: [Hostlist expression](../hostlist/README.md) of hosts sharing this configuration.

### Requirements

- **Platform**: Linux only.
- **Hardware**: Management controllers must support the Redfish API.
