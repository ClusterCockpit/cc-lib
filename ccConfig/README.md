<!--
---
title: ClusterCockpit configuration component
description: Description of ClusterCockpit's configuration component
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccConfig/_index.md
---
-->

# ccConfig interface

The ccConfig component is an approach to read in JSON configuration files in the
ClusterCockpit ecosystem. All configuration files have the following structure:

```json
{
    "main" : {
        // Main configuration for component
    },
    "foo" : {
        // Configuration for the sub-component 'foo'
    }
    "bar" : "bar.json"
}
```

After initializing the ccConfig component with `Init(filename)`, the individual
configuration sections can be retrieved with `GetPackageConfig("key")`. It always
returns the content as `json.RawMessage` and has to be converted as needed.
