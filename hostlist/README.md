<!--
---
title: Hostlist expansion
description: Package to expand hostlists like 'n[0-1],m[2-3]'
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/hostlist/_index.md
---
-->

# Hostlist

The `hostlist` package provides functionality to expand compact hostlist specifications into individual host names. This is particularly useful for cluster computing environments where hosts are often specified using range notation.

## Features

- **Compact notation**: Express multiple hosts concisely using range syntax
- **Zero-padding preservation**: Maintains leading zeros in numeric ranges
- **Automatic sorting**: Results are alphabetically sorted
- **Deduplication**: Automatically removes duplicate host names
- **Flexible syntax**: Supports multiple ranges, indices, and optional suffixes

## Installation

```go
import "github.com/ClusterCockpit/cc-lib/v2/hostlist"
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/ClusterCockpit/cc-lib/v2/hostlist"
)

func main() {
    // Expand a simple range
    hosts, err := hostlist.Expand("node[1-3]")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(hosts)
    // Output: [node1 node2 node3]
}
```

## Syntax

### Basic Patterns

| Pattern | Expands to | Description |
|---------|------------|-------------|
| `n1` | `[n1]` | Single node |
| `n[1-3]` | `[n1, n2, n3]` | Numeric range |
| `n[01-03]` | `[n01, n02, n03]` | Zero-padded range |
| `n[1,3,5]` | `[n1, n3, n5]` | Specific indices |
| `n[1-2,5]` | `[n1, n2, n5]` | Mixed ranges and indices |

### Advanced Patterns

| Pattern | Expands to | Description |
|---------|------------|-------------|
| `n[1-2]-ib` | `[n1-ib, n2-ib]` | Range with suffix |
| `n[1-2],m[3-4]` | `[m3, m4, n1, n2]` | Multiple host groups (sorted) |
| `n1,n1,n2` | `[n1, n2]` | Duplicates removed |
| ` n1 , n2 ` | `[n1, n2]` | Whitespace trimmed |

### Syntax Rules

- **Brackets**: Ranges must be enclosed in square brackets `[]`
- **Range format**: Use hyphen for ranges: `[start-end]`
- **Multiple items**: Separate with commas: `[1,2,5-7]`
- **One range per host**: Only one `[]` specification allowed per host expression
- **Valid characters**: Host names can contain: `a-z`, `A-Z`, `0-9`, `-`
- **Range order**: Start value must be ≤ end value
- **Zero-padding**: Preserved when start and end have the same width

## Usage Examples

### Example 1: Expanding Compute Nodes

```go
hosts, err := hostlist.Expand("compute[001-128]")
if err != nil {
    log.Fatal(err)
}
// Returns: [compute001, compute002, ..., compute128]
fmt.Printf("Total hosts: %d\n", len(hosts))
```

### Example 2: Multiple Node Types

```go
hosts, err := hostlist.Expand("login[1-2],compute[1-4],gpu[1-2]")
if err != nil {
    log.Fatal(err)
}
// Returns: [compute1, compute2, compute3, compute4, gpu1, gpu2, login1, login2]
// Note: Results are sorted alphabetically
```

### Example 3: Network Interfaces

```go
// InfiniBand interfaces
ibHosts, _ := hostlist.Expand("node[1-4]-ib")
// Returns: [node1-ib, node2-ib, node3-ib, node4-ib]

// Ethernet interfaces  
ethHosts, _ := hostlist.Expand("node[1-4]-eth0")
// Returns: [node1-eth0, node2-eth0, node3-eth0, node4-eth0]
```

### Example 4: Error Handling

```go
// Invalid: decreasing range
_, err := hostlist.Expand("node[5-1]")
if err != nil {
    fmt.Println(err)
    // Output: single range start is greater than end: 5-1
}

// Invalid: forbidden character
_, err = hostlist.Expand("node@1")
if err != nil {
    fmt.Println(err)
    // Output: not a hostlist: @1
}

// Invalid: malformed range
_, err = hostlist.Expand("node[1-2-3]")
if err != nil {
    fmt.Println(err)
    // Output: not at hostlist range: [1-2-3]
}
```

## Invalid Specifications

The following patterns will return an error:

- **Multiple hyphens in range**: `[1-2-3]` ❌
- **Decreasing ranges**: `[5-1]` ❌
- **Invalid characters**: `node@1`, `node$1` ❌
- **Nested brackets**: `node[[1-2]]` ❌

## Performance Considerations

- The function uses in-place deduplication for memory efficiency
- Results are sorted using Go's standard `sort.Strings()` (O(n log n))
- Zero-padding detection is performed per range for optimal formatting
- Large ranges (e.g., `[1-10000]`) are expanded efficiently

## Common Use Cases

### Configuration Files

```json
{
  "host_list": "node[01-16]",
  "username": "admin",
  "endpoint": "https://%h-bmc.example.com"
}
```

The `%h` placeholder is typically replaced with each expanded hostname by the calling application.

### SLURM/PBS Job Scripts

```bash
# Expand node list for parallel jobs
NODES=$(echo "node[1-8]" | your-expand-tool)
```

### Monitoring Systems

Used in ClusterCockpit receivers (IPMI, Redfish) to specify which hosts to monitor:

```go
hostList, err := hostlist.Expand(clientConfigJSON.HostList)
if err != nil {
    return fmt.Errorf("failed to parse host list: %v", err)
}
for _, host := range hostList {
    // Monitor each host
}
```

## API Reference

For detailed API documentation, see the [godoc](https://pkg.go.dev/github.com/ClusterCockpit/cc-lib/v2/hostlist).

### Main Function

```go
func Expand(in string) (result []string, err error)
```

Converts a compact hostlist specification into a slice of individual host names. Results are sorted alphabetically and deduplicated.

## Testing

Run the test suite:

```bash
go test -v github.com/ClusterCockpit/cc-lib/v2/hostlist
```

Check test coverage:

```bash
go test -cover github.com/ClusterCockpit/cc-lib/v2/hostlist
```

## License

Copyright (C) NHR@FAU, University Erlangen-Nuremberg.  
Licensed under the MIT License. See LICENSE file for details.

## Contributors

- Holger Obermaier (NHR@KIT)
