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

The `eecpt` receiver can be used receive metrics from the EECPT instrumentation library

### Configuration structure

```json
{
  "<name>": {
    "type": "eecpt",
    "address" : "",
    "port" : "8080",
    "path" : "/write",
    "idle_timeout": "120s",
    "username": "myUser",
    "password": "myPW",
    "analysis_buffer_size": 10,
    "analysis_interval": "5m",
    "analysis_metric": "region_metric"
  }
}
```

The EECPT library sends data in the format:
```
non_blocking_calls,type=node,stype=job,stype-id=<jobid>,application=<appname>,hostname=<hostname> value=<value>,rank=<0,1,2,3,...> <timestamp>
```
