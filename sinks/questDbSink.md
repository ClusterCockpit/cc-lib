<!--
---
title: Message sink to QuestDB
description: Message sink for QuestDB endpoints
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/sinks/questDBSink.md
---
-->

## `questdb` sink

The `questdb` sink sends metrics to the timeseries database QuestDB

### Configuration structure

```json
{
  "<name>": {
    "type": "questdb",
    "address" : "hostname:port",
    "username": "myUser",
    "password": "myPW",
    "bearer_token": "myBearerToken",
    "auto_flush_interval": "5s",
    "auto_flush_rows": 1000,
    "process_messages" : {
      "see" : "docs of message processor for valid fields"
    }
  }
}
```

- `type`: makes the sink an `questdb` sink
- `address`: The hostname and port to connect for QuestDBs REST API (default `localhost:9000`)
- `username`: username for basic authentication
- `password`: password for basic authentication
- `bearer_token`: authentication with bearer token in HTTP header
- `auto_flush_interval`: interval at which the sender automatically flushes its buffer (default `5s`)
- `auto_flush_rows`: number of rows after which the sender automatically flushes its buffer
