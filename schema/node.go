// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

type SchedulerState string

const (
	NodeStateAllocated SchedulerState = "allocated"
	NodeStateReserved  SchedulerState = "reserved"
	NodeStateIdle      SchedulerState = "idle"
	NodeStateMixed     SchedulerState = "mixed"
	NodeStateDown      SchedulerState = "down"
	NodeStateUnknown   SchedulerState = "unknown"
)

type MonitoringState string

const (
	MonitoringStateFull    MonitoringState = "full"
	MonitoringStatePartial MonitoringState = "partial"
	MonitoringStateFailed  MonitoringState = "failed"
)

type Node struct {
	Hostname        string            `json:"hostname"`
	Cluster         string            `json:"cluster"`
	SubCluster      string            `json:"subCluster"`
	MetaData        map[string]string `json:"metaData"`
	NodeState       SchedulerState    `json:"nodeState"`
	HealthState     MonitoringState   `json:"healthState"`
	CpusAllocated   int               `json:"cpusAllocated"`
	MemoryAllocated int               `json:"memoryAllocated"`
	GpusAllocated   int               `json:"gpusAllocated"`
	JobsRunning     int               `json:"jobsRunning"`
}

type NodePayload struct {
	Hostname        string   `json:"hostname"`
	States          []string `json:"states"`
	CpusAllocated   int      `json:"cpusAllocated"`
	MemoryAllocated int      `json:"memoryAllocated"`
	GpusAllocated   int      `json:"gpusAllocated"`
	JobsRunning     int      `json:"jobsRunning"`
}

type NodeDB struct {
	ID          int64  `json:"id" db:"id"`
	Hostname    string `json:"hostname" db:"hostname" example:"fritz"`
	Cluster     string `json:"cluster" db:"cluster" example:"fritz"`
	SubCluster  string `json:"subCluster" db:"subcluster" example:"main"`
	RawMetaData []byte `json:"-" db:"meta_data"`
}

type NodeStateDB struct {
	ID              int64           `json:"id" db:"id"`
	TimeStamp       int64           `json:"timeStamp" db:"time_stamp" example:"1649723812"`
	NodeState       SchedulerState  `json:"nodeState" db:"node_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"`
	HealthState     MonitoringState `json:"healthState" db:"health_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"`
	CpusAllocated   int             `json:"cpusAllocated" db:"cpus_allocated"`
	MemoryAllocated int             `json:"memoryAllocated" db:"memory_allocated"`
	GpusAllocated   int             `json:"gpusAllocated" db:"gpus_allocated"`
	JobsRunning     int             `json:"jobsRunning" db:"jobs_running" example:"12"`
	NodeID          int64           `json:"_" db:"node_id"`
}
