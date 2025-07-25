// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

type NodeState string

const (
	NodeStateAllocated NodeState = "allocated"
	NodeStateReserved  NodeState = "reserved"
	NodeStateIdle      NodeState = "idle"
	NodeStateMixed     NodeState = "mixed"
	NodeStateDown      NodeState = "down"
	NodeStateUnknown   NodeState = "unknown"
)

type MonitoringState string

const (
	MonitoringStateFull    MonitoringState = "full"
	MonitoringStatePartial MonitoringState = "partial"
	MonitoringStateFailed  MonitoringState = "failed"
)

type Node struct {
	ID              int64             `json:"id" db:"id"`
	TimeStamp       int64             `json:"timeStamp" db:"time_stamp" example:"1649723812"`
	Hostname        string            `json:"hostname" db:"hostname" example:"fritz"`
	Cluster         string            `json:"cluster" db:"cluster" example:"fritz"`
	SubCluster      string            `json:"subCluster" db:"subcluster" example:"main"`
	NodeState       NodeState         `json:"nodeState" db:"node_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"`
	HealthState     MonitoringState   `json:"healthState" db:"health_state" example:"completed" enums:"completed,failed,cancelled,stopped,timeout,out_of_memory"`
	CpusAllocated   int               `json:"cpusAllocated" db:"cpus_allocated"`
	CpusTotal       int               `json:"cpusTotal" db:"cpus_total"`
	MemoryAllocated int               `json:"memoryAllocated" db:"memory_allocated"`
	MemoryTotal     int               `json:"memoryTotal" db:"memory_total"`
	GpusAllocated   int               `json:"gpusAllocated" db:"gpus_allocated"`
	GpusTotal       int               `json:"gpusTotal" db:"gpus_total"`
	JobsRunning     int64             `json:"jobsRunning" db:"jobs_running" example:"12"`
	RawMetaData     []byte            `json:"-" db:"meta_data"`
	MetaData        map[string]string `json:"metaData"`
}
