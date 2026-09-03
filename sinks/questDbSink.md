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
    "use_tls": false,
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
- `use_tls`: Use https instead of http transport protocol

### Credentials from the environment

The credentials above need not be stored in the configuration file. Each of
`username`, `password`, `bearer_token` has two sibling keys that select an alternative source:

- `username_env`: name of an environment variable holding the value
- `username_file`: path to a file holding the value
- `password_env`: name of an environment variable holding the value
- `password_file`: path to a file holding the value
- `bearer_token_env`: name of an environment variable holding the value
- `bearer_token_file`: path to a file holding the value

The environment variable takes precedence over the file, and the file over the
inline value. A named file that cannot be read is an error rather than a silent
fallback, so a stale credential is never used in its place. See the
[`util`](../util/README.md) package for the full rules.
