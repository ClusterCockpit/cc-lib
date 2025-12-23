<!--
---
title: Message receiver for NATS pub-sub networks
description: Message receiver for NATS pub-sub networks
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/nats.md
---
-->


## `nats` receiver

The `nats` receiver subscribes to a [NATS](https://nats.io/) subject to receive metrics in InfluxDB line protocol. It is useful for decoupled metric collection where sources publish to a message bus.

### Configuration Structure

```json
{
  "my_nats_receiver": {
    "type": "nats",
    "address" : "nats-server.example.org",
    "port" : "4222",
    "subject" : "metrics",
    "user": "natsuser",
    "password": "natssecret",
    "nkey_file": "/path/to/nkey_file",
    "process_messages": []
  }
}
```

### Configuration Options

- `type`: Must be `nats`.
- `address`: Hostname or IP of the NATS server (default: `localhost`).
- `port`: Port of the NATS server (default: `4222`).
- `subject`: (Required) The NATS subject to subscribe to.
- `user`: Optional username for authentication.
- `password`: Optional password for authentication.
- `nkey_file`: Optional path to an NKEY credentials file.
- `process_messages`: Optional message processing rules.

### Debugging

You can use the NATS command line client to interact with the server and verify the receiver.

1.  **Check NATS server status**:
    ```bash
    nats --server=nats-server.example.org:4222 server check
    ```

2.  **Monitor all messages**:
    ```bash
    nats --server=nats-server.example.org:4222 sub ">"
    ```

3.  **Publish test metrics**:
    ```bash
    nats --server=nats-server.example.org:4222 pub metrics \
    "myMetric,hostname=myHost,type=hwthread,type-id=0,unit=Hz value=400000i 1694777161164284635
    myMetric,hostname=myHost,type=hwthread,type-id=1,unit=Hz value=400001i 1694777161164284635"
    ```
