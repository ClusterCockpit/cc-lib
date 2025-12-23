<!--
---
title: Message receiver for IPMI endpoints
description: Query metrics from remote IPMI sources
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/ipmi.md
---
-->

## `ipmi` receiver

The `ipmi` receiver uses `ipmi-sensors` from the [FreeIPMI](https://www.gnu.org/software/freeipmi/) project to read IPMI sensor readings and sensor data repository (SDR) information. It is designed for polling metrics from BMCs (Baseboard Management Controllers).

### Configuration Structure

```json
{
  "my_ipmi_receiver": {
    "type": "ipmi",
    "interval": "30s",
    "fanout": 256,
    "username": "admin",
    "password": "password",
    "endpoint": "ipmi-sensors://%h-bmc",
    "exclude_metrics": [ "fan_speed", "voltage" ],
    "process_messages": [],
    "client_config": [
      {
        "host_list": "node[01-04]"
      },
      {
        "host_list": "node[05-08]",
        "driver_type": "LAN_2_0",
        "cli_options": [ "--workaround-flags=..." ],
        "password": "different_password"
      }
    ]
  }
}
```

### Global Configuration Options

- `type`: Must be `ipmi`.
- `interval`: How often to poll the IPMI sensors (default: `30s`).
- `fanout`: Maximum number of simultaneous IPMI connections (default: `64`).
- `process_messages`: Optional message processing rules.

### Global and Per-Device Options

These settings can be defined globally and overridden in `client_config`:

- `endpoint`: URL/Template for the IPMI device. `%h` is replaced by the hostname (e.g., `ipmi-sensors://%h-bmc`).
- `username`: Username for authentication.
- `password`: Password for authentication.
- `driver_type`: IPMI driver type (default: `LAN_2_0`).
- `exclude_metrics`: List of metrics to exclude (e.g., `fan_speed`, `voltage`, `temperature`, `power`, `utilization`).

### Per-Device Options (`client_config`)

- `host_list`: [Hostlist expression](../hostlist/README.md) of hosts sharing this configuration.
- `cli_options`: Additional command line options passed to `ipmi-sensors`.

### Requirements

- **Platform**: Linux only.
- **Tools**: `ipmi-sensors` must be installed and available in the PATH.
- **Permissions**: The user running the collector must have permission to execute `ipmi-sensors`.
