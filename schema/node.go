// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

// SchedulerState represents the current state of a node in the HPC job scheduler.
// States typically reflect SLURM/PBS node states.
type SchedulerState string

const (
	NodeStateAllocated SchedulerState = "allocated" // Node is fully allocated to jobs
	NodeStateReserved  SchedulerState = "reserved"  // Node is reserved but not yet allocated
	NodeStateIdle      SchedulerState = "idle"      // Node is available for jobs
	NodeStateMixed     SchedulerState = "mixed"     // Node is partially allocated
	NodeStateDown      SchedulerState = "down"      // Node is down/offline
	NodeStateUnknown   SchedulerState = "unknown"   // Node state unknown
)

// MonitoringState indicates the health monitoring status of a node.
// Reflects whether metric collection is working correctly.
type MonitoringState string

const (
	MonitoringStateFull    MonitoringState = "full"    // All metrics being collected successfully
	MonitoringStatePartial MonitoringState = "partial" // Some metrics missing
	MonitoringStateFailed  MonitoringState = "failed"  // Metric collection failing
)

// Node represents the current state and resource utilization of a compute node.
//
// Combines scheduler state with monitoring health and current resource allocation.
// Used for displaying node status in dashboards and tracking node utilization.
type Node struct {
	Hostname        string            `json:"hostname"`        // Node hostname
	Cluster         string            `json:"cluster"`         // Cluster name
	SubCluster      string            `json:"subCluster"`      // Subcluster name
	MetaData        map[string]string `json:"metaData"`        // Additional metadata
	NodeState       SchedulerState    `json:"nodeState"`       // Scheduler/resource manager state
	HealthState     MonitoringState   `json:"healthState"`     // Monitoring system health
	CpusAllocated   int               `json:"cpusAllocated"`   // Number of allocated CPUs
	MemoryAllocated int               `json:"memoryAllocated"` // Allocated memory in MB
	GpusAllocated   int               `json:"gpusAllocated"`   // Number of allocated GPUs
	JobsRunning     int               `json:"jobsRunning"`     // Number of jobs running on this node
}

// NodePayload is the request body format for the node state REST API.
// Used when updateing node states from external monitoring or scheduler systems.
type NodePayload struct {
	Hostname        string   `json:"hostname"`        // Node hostname
	States          []string `json:"states"`          // State strings (flexible format)
	CpusAllocated   int      `json:"cpusAllocated"`   // Number of allocated CPUs
	MemoryAllocated int      `json:"memoryAllocated"` // Allocated memory in MB
	GpusAllocated   int      `json:"gpusAllocated"`   // Number of allocated GPUs
	JobsRunning     int      `json:"jobsRunning"`     // Number of running jobs
}

// NodeDB is the database model for the node table.
// Stores static node configuration and metadata.
type NodeDB struct {
	ID          int64  `json:"id" db:"id"`                                // Database ID
	Hostname    string `json:"hostname" db:"hostname" example:"fritz"`    // Node hostname
	Cluster     string `json:"cluster" db:"cluster" example:"fritz"`      // Cluster name
	SubCluster  string `json:"subCluster" db:"subcluster" example:"main"` // Subcluster name
	RawMetaData []byte `json:"-" db:"meta_data"`                          // Metadata as JSON blob
}

// NodeStateDB is the database model for the node_state table.
// Stores time-stamped snapshots of node state and resource allocation.
type NodeStateDB struct {
	ID              int64           `json:"id" db:"id"`                                                                                                         // Database ID
	TimeStamp       int64           `json:"timeStamp" db:"time_stamp" example:"1649723812"`                                                                     // Unix timestamp
	NodeState       SchedulerState  `json:"nodeState" db:"node_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"`     // Scheduler state
	HealthState     MonitoringState `json:"healthState" db:"health_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"` // Monitoring health
	CpusAllocated   int             `json:"cpusAllocated" db:"cpus_allocated"`                                                                                  // Allocated CPUs
	MemoryAllocated int             `json:"memoryAllocated" db:"memory_allocated"`                                                                              // Allocated memory (MB)
	GpusAllocated   int             `json:"gpusAllocated" db:"gpus_allocated"`                                                                                  // Allocated GPUs
	JobsRunning     int             `json:"jobsRunning" db:"jobs_running" example:"12"`                                                                         // Running jobs
	NodeID          int64           `json:"_" db:"node_id"`                                                                                                     // Foreign key to NodeDB
}
