// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package schema provides core data structures and types for the ClusterCockpit system.
//
// This package defines the fundamental schemas used throughout ClusterCockpit for representing
// HPC job metadata, cluster configurations, performance metrics, user authentication, and
// validation utilities.
//
// Key components:
//
// Job Data Structures:
//   - Job: Complete metadata for HPC jobs including resources, state, and statistics
//   - JobMetric: Performance metrics data with time series and statistics
//   - JobState: Enumeration of possible job states (running, completed, failed, etc.)
//
// Cluster Configuration:
//   - Cluster: HPC cluster definition with subclusters and metric configuration
//   - SubCluster: Partition of a cluster with specific hardware topology
//   - Topology: Hardware topology mapping (nodes, sockets, cores, accelerators)
//
// Metrics and Statistics:
//   - MetricScope: Hierarchical metric scopes (node, socket, core, hwthread, accelerator)
//   - Series: Time series data for metrics with statistics
//   - JobStatistics: Statistical aggregations (min, avg, max) for job metrics
//
// User Management:
//   - User: User account with roles, projects, and authentication information
//   - Role: Authorization levels (admin, support, manager, user, api, anonymous)
//   - AuthSource: Authentication source types (local, LDAP, token, OIDC)
//
// Validation:
//   - Validate: JSON schema validation for job metadata, job data, and cluster configs
//   - Kind: Enumeration of schema types for validation
//
// Special Types:
//   - Float: Custom float64 wrapper that handles NaN as JSON null for efficient metric storage
//   - Node: Node state information including scheduler and monitoring states
//
// The types in this package are designed to be serialized to/from JSON and are used
// across REST APIs, GraphQL interfaces, and internal data processing pipelines.
package schema
