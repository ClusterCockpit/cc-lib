<!--
---
title: Message receiver for HTTP
description: Receiving messages over HTTP from remote sources
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/http.md
---
-->

## `http` receiver

The `http` receiver accepts metrics through HTTP POST requests. It is commonly used to receive data from remote collectors or applications that can push metrics in InfluxDB line protocol.

### Configuration Structure

```json
{
  "my_http_receiver": {
    "type": "http",
    "address" : "0.0.0.0",
    "port" : "8080",
    "path" : "/write",
    "idle_timeout": "120s",
    "keep_alives_enabled": true,
    "username": "myUser",
    "password": "myPW",
    "process_messages": []
  }
}
```

### Configuration Options

- `type`: Must be `http`.
- `address`: IP address to listen on (default: empty for all interfaces).
- `port`: Port to listen on (default: `8080`).
- `path`: URL path for the write endpoint (e.g., `/write`).
- `idle_timeout`: Maximum idle time for keep-alive connections (default: `120s`).
- `keep_alives_enabled`: Whether to enable HTTP keep-alives (default: `true`).
- `username`: Optional username for basic authentication.
- `password`: Optional password for basic authentication.
- `process_messages`: Optional message processing rules.

The HTTP endpoint listens at `http://<address>:<port>/<path>`.

### Ingress Format

The receiver expects data in [InfluxDB line protocol](https://docs.influxdata.com/influxdb/v2.7/reference/syntax/line-protocol/). Multiple lines can be sent in a single POST request.

### Debugging

You can use `curl` to test the receiver:

```bash
curl http://localhost:8080/write \
  --user "myUser:myPW" \
  --data \
"myMetric,hostname=myHost,type=hwthread,type-id=0,unit=Hz value=400000i 1694777161164284635
myMetric,hostname=myHost,type=hwthread,type-id=1,unit=Hz value=400001i 1694777161164284635"
```
