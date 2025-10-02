<!--
---
title: Startup signalling
description: Startup signalling for ClusterCockpit
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccStartup/_index.md
---
-->

# ccStartup

ccStartup is used to provide the current node topology to some central endpoint either through HTTP or NATS.

# Configuration

The configuration file for the startup contains the targets where the topology should be sent to.

```json
{
    "http": {
        "url": "http://localhost:8080/my_startup_endpoint",
        "auth_token": "my-secret-login-token"
    },
    "nats": {
        "url": "nats://localhost:4222",
        "nkey_file:": "/path/to/nkey/file/if/any",
        "subject": "my_startup_subject"
    }
}
```

- `http.url`: Target URL for the HTTP POST request containing the node topology as JSON
- `http.auth_token`: JSON Web token or <username>:<password>
- `nats.url`: NATS server URL
- `nats.subject`: NATS subject where to publish the topology as JSON
- `nats.nkey_file`: Path to NKey file for authentification
