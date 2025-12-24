<!--
---
title: Message scraper for Prometheus
description: Message scraper for Prometheus monitoring endpoints
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/receivers/prometheus.md
---
-->


## `prometheus` receiver

The `prometheus` receiver scrapes metrics from a single Prometheus-compatible endpoint. It periodically makes HTTP GET requests to the configured endpoint and parses the response.

### Configuration Structure

```json
{
  "my_prometheus_receiver": {
    "type": "prometheus",
    "address" : "prometheus-client.example.org",
    "port" : "9100",
    "path" : "/metrics",
    "interval": "15s",
    "ssl" : false,
    "process_messages": []
  }
}
```

### Configuration Options

- `type`: Must be `prometheus`.
- `address`: Hostname or IP of the Prometheus agent.
- `port`: Port of the Prometheus agent.
- `path`: Path to the Prometheus endpoint (default: `/metrics`).
- `interval`: Scrape interval (default: `5s`).
- `ssl`: Whether to use HTTPS (default: `false`).
- `process_messages`: Optional message processing rules.

The receiver requests data from `http(s)://<address>:<port>/<path>`.

### Implementation Notes

This receiver does not use the official Prometheus client library. Instead, it performs simple HTTP requests and parses the Prometheus text format manually to minimize dependencies.
