<!--
---
title: Message sink to QuestDB
description: Message sink for QuestDB endpoints
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/sinks/http.md
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
