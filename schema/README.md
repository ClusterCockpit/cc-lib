# Schema Package

The `schema` package provides core data structures and types for the ClusterCockpit system, including HPC job metadata, cluster configurations, performance metrics, user authentication, and validation utilities.

## Overview

This package defines the fundamental schemas used throughout ClusterCockpit for representing:
- **Job Data**: Complete metadata and metrics for HPC jobs
- **Cluster Configuration**: Hardware topology and metric collection settings
- **Performance Metrics**: Time series data with statistical aggregations
- **User Management**: Authentication and authorization
- **Validation**: JSON schema validation for data integrity

## Key Components

### Job Structures

#### `Job`
The central data structure containing all information about an HPC job:
- Identification (cluster, job ID, user, project)
- Resources (nodes, cores, accelerators, memory)
- Timing (submission, start, duration)
- State (current job state, monitoring status)
- Metrics (performance statistics, time series data)
- Metadata (tags, energy footprint)

#### `JobMetric`
Performance metric data with time series from individual hardware components and aggregated statistics.

#### `JobState`
Enumeration of job execution states matching common HPC schedulers (SLURM, PBS).

### Cluster Configuration

#### `Cluster`
Complete HPC cluster definition with subclusters and metric collection settings.

#### `SubCluster`
Homogeneous partition within a cluster sharing identical hardware configuration.

#### `Topology`
Hardware topology mapping showing relationships between nodes, sockets, cores, hardware threads, and accelerators.

### Metrics and Statistics

#### `MetricScope`
Hierarchical levels for metric measurement:
```
node > socket > memoryDomain > core > hwthread
(accelerator is a special scope at hwthread level)
```

#### `Series`
Time series of metric measurements from a single source (node, core, etc.) with min/avg/max statistics.

#### `Float`
Custom float64 type with special NaN handling:
- NaN values serialize as JSON `null`
- JSON `null` deserializes to NaN
- Avoids pointer overhead for nullable metrics
- Compatible with both JSON and GraphQL

### User Management

#### `User`
User account with authentication and authorization information.

#### `Role`
Authorization hierarchy:
```
Anonymous < Api < User < Manager < Support < Admin
```

#### `AuthSource`
Authentication backends: LocalPassword, LDAP, Token, OIDC

### Validation

#### `Validate(kind Kind, r io.Reader) error`
Validates JSON data against embedded JSON schemas:
- `Meta`: Job metadata structure
- `Data`: Job metric data
- `ClusterCfg`: Cluster configuration

## Usage Examples

### Validating Cluster Configuration

```go
import (
    "bytes"
    "github.com/ClusterCockpit/cc-lib/schema"
)

// Validate cluster.json against schema
err := schema.Validate(schema.ClusterCfg, bytes.NewReader(clusterJSON))
if err != nil {
    log.Fatal("Invalid cluster configuration:", err)
}
```

### Working with Metrics

```go
// Create job metric data
jobMetric := &schema.JobMetric{
    Unit:     schema.Unit{Base: "FLOP/s", Prefix: "G"},
    Timestep: 60,
    Series: []schema.Series{
        {
            Hostname: "node001",
            Data:     []schema.Float{1.5, 2.0, 1.8, schema.NaN}, // NaN for missing data
            Statistics: schema.MetricStatistics{
                Min: 1.5,
                Avg: 1.77,
                Max: 2.0,
            },
        },
    },
}

// Add aggregated statistics
jobMetric.AddStatisticsSeries()
```

### User Role Checking

```go
user := &schema.User{
    Username: "alice",
    Roles:    []string{"user", "manager"},
}

// Check if user has manager role
if user.HasRole(schema.RoleManager) {
    // Grant project-level access
}

// Check for admin or support
if user.HasAnyRole([]schema.Role{schema.RoleAdmin, schema.RoleSupport}) {
    // Grant elevated privileges
}
```

### Topology Navigation

```go
// Get sockets used by a job's hardware threads
hwthreads := []int{0, 1, 2, 3, 20, 21, 22, 23}
sockets, exclusive := topology.GetSocketsFromHWThreads(hwthreads)

if exclusive {
    fmt.Printf("Job has exclusive access to sockets: %v\n", sockets)
} else {
    fmt.Printf("Job shares sockets: %v\n", sockets)
}
```

## Database Models

The package includes database models for persistent storage:

- `NodeDB`: Static node configuration
- `NodeStateDB`: Time-stamped node state snapshots
- `Job`: Contains both API fields and raw database fields (Raw*)

Raw database fields (`RawResources`, `RawMetaData`, etc.) store JSON blobs that are decoded into typed fields when loaded from the database.

## JSON Schema Files

Embedded JSON schemas in `schemas/` directory:
- `cluster.schema.json`: Cluster configuration validation
- `job-meta.schema.json`: Job metadata validation
- `job-data.schema.json`: Job metric data validation
- `job-metric-data.schema.json`: Individual metric validation
- `job-metric-statistics.schema.json`: Metric statistics validation
- `unit.schema.json`: Unit of measurement validation

## Performance Considerations

### Float Type
The custom `Float` type avoids pointer overhead for nullable metrics. In large-scale metric storage with millions of data points, this provides significant memory savings compared to `*float64`.

### Series Marshaling
The `Series.MarshalJSON()` method provides optimized JSON serialization with fewer allocations, important for REST API performance when returning large metric datasets.

## Related Packages

- `ccLogger`: Logging utilities used for validation warnings
- `util`: Helper functions for statistics (median calculation)
- `jsonschema/v5`: JSON schema validation implementation

## Testing

Run tests:
```bash
go test ./schema/... -v
```

Check test coverage:
```bash
go test ./schema/... -cover
```

View godoc:
```bash
go doc -all schema
```
