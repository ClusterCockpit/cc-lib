// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Job represents complete metadata for an HPC job in ClusterCockpit.
//
// This is the central data structure containing all information about a job including:
// - Identification: cluster, job ID, user, project
// - Resources: nodes, cores, accelerators, memory
// - Timing: submission, start time, duration
// - State: current job state and monitoring status
// - Metrics: performance statistics and time series data
// - Metadata: tags, energy footprint, custom metadata
//
// The RawX fields are used for database  serialization of complex nested structures
// that are stored as JSON blobs in the database and decoded into their respective
// typed fields (Resources, EnergyFootprint, Footprint, MetaData) when loaded.
//
// Job model
// @Description Information of a HPC job.
type Job struct {
	Cluster            string             `json:"cluster" db:"cluster" example:"fritz"`
	SubCluster         string             `json:"subCluster" db:"subcluster" example:"main"`
	Partition          string             `json:"partition,omitempty" db:"cluster_partition" example:"main"`
	Project            string             `json:"project" db:"project" example:"abcd200"`
	User               string             `json:"user" db:"hpc_user" example:"abcd100h"`
	Shared             string             `json:"shared" db:"shared" enums:"none,single_user,multi_user"`
	State              JobState           `json:"jobState" db:"job_state" example:"completed" enums:"boot_fail,cancelled,completed,deadline,failed,node_fail,out-of-memory,pending,preempted,running,suspended,timeout"`
	Tags               []*Tag             `json:"tags,omitempty"`
	RawEnergyFootprint []byte             `json:"-" db:"energy_footprint"`
	RawFootprint       []byte             `json:"-" db:"footprint"`
	RawMetaData        []byte             `json:"-" db:"meta_data"`
	RawResources       []byte             `json:"-" db:"resources"`
	Resources          []*Resource        `json:"resources"`
	EnergyFootprint    map[string]float64 `json:"energyFootprint"`
	Footprint          map[string]float64 `json:"footprint"`
	MetaData           map[string]string  `json:"metaData"`
	ConcurrentJobs     JobLinkResultList  `json:"concurrentJobs"`
	Energy             float64            `json:"energy" db:"energy"`
	ArrayJobID         int64              `json:"arrayJobId,omitempty" db:"array_job_id" example:"123000"`
	Walltime           int64              `json:"walltime,omitempty" db:"walltime" example:"86400" minimum:"1"`
	RequestedMemory    int64              `json:"requestedMemory,omitempty" db:"requested_memory" example:"128000" minimum:"1"` // in MB
	JobID              int64              `json:"jobId" db:"job_id" example:"123000"`
	Duration           int32              `json:"duration" db:"duration" example:"43200" minimum:"1"`
	SMT                int32              `json:"smt,omitempty" db:"smt" example:"4"`
	MonitoringStatus   int32              `json:"monitoringStatus,omitempty" db:"monitoring_status" example:"1" minimum:"0" maximum:"3"`
	NumAcc             int32              `json:"numAcc,omitempty" db:"num_acc" example:"2" minimum:"1"`
	NumHWThreads       int32              `json:"numHwthreads,omitempty" db:"num_hwthreads" example:"20" minimum:"1"`
	NumNodes           int32              `json:"numNodes" db:"num_nodes" example:"2" minimum:"1"`
	Statistics         JobStatisticsSet   `json:"statistics"`
	ID                 *int64             `json:"id,omitempty" db:"id"`
	SubmitTime         int64              `json:"submitTime,omitempty" db:"submit_time" example:"1649723812"`
	StartTime          int64              `json:"startTime" db:"start_time" example:"1649723812"`
}

// JobLink represents a lightweight reference to a job, typically used for linking related jobs.
// Used to track concurrent jobs or job relationships without including full job metadata.
type JobLink struct {
	ID    int64 `json:"id"`    // Internal database ID
	JobID int64 `json:"jobId"` // The job's external job ID
}

// JobLinkResultList holds a paginated list of job links with a total count.
// Typically used for API responses that return lists of related jobs.
type JobLinkResultList struct {
	Items []*JobLink `json:"items"` // List of job links
	Count int        `json:"count"` // Total count of available items
}

const (
	MonitoringStatusDisabled            int32 = 0
	MonitoringStatusRunningOrArchiving  int32 = 1
	MonitoringStatusArchivingFailed     int32 = 2
	MonitoringStatusArchivingSuccessful int32 = 3
)

// var JobDefaults Job = Job{
// 	Exclusive:        1,
// 	MonitoringStatus: MonitoringStatusRunningOrArchiving,
// }

// Unit represents a unit of measurement with optional SI prefix.
//
// Examples:
//   - {Base: "B/s", Prefix: "G"} = GB/s (gigabytes per second)
//   - {Base: "F/s", Prefix: "T"} = TF/s (teraflops per second)
//   - {Base: "", Prefix: ""}     = dimensionless (e.g., CPU load)
type Unit struct {
	Base   string `json:"base"`             // Base unit (e.g., "B/s", "F/s", "W")
	Prefix string `json:"prefix,omitempty"` // SI prefix (e.g., "G", "M", "K", "T")
}

// JobStatistics model
// @Description Specification for job metric statistics.
type JobStatistics struct {
	Unit Unit    `json:"unit"`
	Avg  float64 `json:"avg" example:"2500" minimum:"0"` // Job metric average
	Min  float64 `json:"min" example:"2000" minimum:"0"` // Job metric minimum
	Max  float64 `json:"max" example:"3000" minimum:"0"` // Job metric maximum
}

// StatsGroupInstance is one named element of an array-valued statistics group
// (a single filesystem, later a single interconnect). Its per-instance metrics
// are flat JobStatistics (the job-meta schema does not scope filesystem stats).
type StatsGroupInstance struct {
	Name    string                   `json:"name"`
	Type    string                   `json:"type"`
	Metrics map[string]JobStatistics `json:"-"` // flattened onto the instance object by MarshalJSON
}

// StatsGroup is an array-valued statistics group identified by its top-level
// JSON key (e.g. "filesystems"). Instance order is preserved.
type StatsGroup struct {
	Key       string
	Instances []StatsGroupInstance
}

// JobStatisticsSet is the value of Job.Statistics. Metrics holds the flat
// per-metric aggregates; Groups holds array-valued groups (filesystems, and
// later interconnects) whose members each carry their own statistics. Custom
// (Un)MarshalJSON keep the job-meta "statistics" layout: flat metrics as
// objects plus each group as a nested array.
type JobStatisticsSet struct {
	Metrics map[string]JobStatistics
	Groups  []StatsGroup
}

// MarshalJSON renders the job-meta "statistics" layout: each flat metric as
// {unit,avg,min,max}, and each group as an array of
// {name, type, <metric>: {unit,avg,min,max}, ...} instances.
func (s JobStatisticsSet) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(s.Metrics)+len(s.Groups))

	for name, stat := range s.Metrics {
		b, err := json.Marshal(stat)
		if err != nil {
			return nil, err
		}
		out[name] = b
	}

	for _, group := range s.Groups {
		arr := make([]json.RawMessage, 0, len(group.Instances))
		for _, inst := range group.Instances {
			obj := make(map[string]json.RawMessage, len(inst.Metrics)+2)
			name, err := json.Marshal(inst.Name)
			if err != nil {
				return nil, err
			}
			obj["name"] = name
			typ, err := json.Marshal(inst.Type)
			if err != nil {
				return nil, err
			}
			obj["type"] = typ
			for metric, stat := range inst.Metrics {
				b, err := json.Marshal(stat)
				if err != nil {
					return nil, err
				}
				obj[metric] = b
			}
			b, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			arr = append(arr, b)
		}
		b, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		out[group.Key] = b
	}

	return json.Marshal(out)
}

// UnmarshalJSON parses the job-meta "statistics" layout into a JobStatisticsSet.
// Array-valued group keys (registered, or fallback: value starts with '[') are
// decoded into Groups; all other keys into flat Metrics.
func (s *JobStatisticsSet) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	s.Metrics = make(map[string]JobStatistics, len(raw))
	s.Groups = nil

	for key, msg := range raw {
		if IsMetricGroupKey(key) || firstToken(msg) == '[' {
			group := StatsGroup{Key: key}
			var items []map[string]json.RawMessage
			if err := json.Unmarshal(msg, &items); err != nil {
				return err
			}
			for _, item := range items {
				inst := StatsGroupInstance{Metrics: make(map[string]JobStatistics, len(item))}
				for field, fmsg := range item {
					switch field {
					case "name":
						if err := json.Unmarshal(fmsg, &inst.Name); err != nil {
							return err
						}
					case "type":
						if err := json.Unmarshal(fmsg, &inst.Type); err != nil {
							return err
						}
					default:
						var stat JobStatistics
						if err := json.Unmarshal(fmsg, &stat); err != nil {
							return err
						}
						inst.Metrics[field] = stat
					}
				}
				group.Instances = append(group.Instances, inst)
			}
			s.Groups = append(s.Groups, group)
			continue
		}

		var stat JobStatistics
		if err := json.Unmarshal(msg, &stat); err != nil {
			return err
		}
		s.Metrics[key] = stat
	}

	return nil
}

// AddGroupInstance appends a named statistics instance to the group identified
// by groupKey, creating the group if necessary.
func (s *JobStatisticsSet) AddGroupInstance(groupKey, name, typ string, metrics map[string]JobStatistics) {
	inst := StatsGroupInstance{Name: name, Type: typ, Metrics: metrics}
	for i := range s.Groups {
		if s.Groups[i].Key == groupKey {
			s.Groups[i].Instances = append(s.Groups[i].Instances, inst)
			return
		}
	}
	s.Groups = append(s.Groups, StatsGroup{Key: groupKey, Instances: []StatsGroupInstance{inst}})
}

// Tag model
// @Description Defines a tag using name and type.
type Tag struct {
	Type  string `json:"type" db:"tag_type" example:"Debug"`
	Name  string `json:"name" db:"tag_name" example:"Testjob"`
	Scope string `json:"scope" db:"tag_scope" example:"global"`
	ID    int64  `json:"id" db:"id"`
}

// Resource represents the hardware resources assigned to a job on a single compute node.
//
// A job typically uses multiple Resource entries, one for each allocated node.
// HWThreads lists the specific hardware thread IDs allocated, allowing for precise
// CPU pinning analysis. Accelerators lists assigned GPU/accelerator IDs.
//
// Resource model
// @Description A resource used by a job
type Resource struct {
	Hostname      string   `json:"hostname"`                // Node hostname
	Configuration string   `json:"configuration,omitempty"` // Optional configuration identifier
	HWThreads     []int    `json:"hwthreads,omitempty"`     // Allocated hardware thread IDs
	Accelerators  []string `json:"accelerators,omitempty"`  // Allocated accelerator IDs (e.g., GPU IDs)
}

// JobState represents the execution state of an HPC job.
// Valid states match common HPC scheduler states (SLURM, PBS, etc.).
type JobState string

const (
	JobStateBootFail    JobState = "boot_fail"
	JobStateCancelled   JobState = "cancelled"
	JobStateCompleted   JobState = "completed"
	JobStateDeadline    JobState = "deadline"
	JobStateFailed      JobState = "failed"
	JobStateNodeFail    JobState = "node_fail"
	JobStateOutOfMemory JobState = "out_of_memory"
	JobStatePending     JobState = "pending"
	JobStatePreempted   JobState = "preempted"
	JobStateRunning     JobState = "running"
	JobStateSuspended   JobState = "suspended"
	JobStateTimeout     JobState = "timeout"
)

func (j Job) GoString() string {
	return fmt.Sprintf("Job{ID:%d, StartTime:%d, JobID:%v, BaseJob:%v}",
		j.ID, j.StartTime, j.JobID, j)
}

func (e *JobState) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("SCHEMA/JOB > enums must be strings")
	}

	*e = JobState(str)
	if !e.Valid() {
		return errors.New("SCHEMA/JOB > invalid job state")
	}

	return nil
}

func (e JobState) MarshalGQL(w io.Writer) {
	fmt.Fprintf(w, "\"%s\"", e)
}

func (e JobState) Valid() bool {
	return e == JobStateRunning ||
		e == JobStateCompleted ||
		e == JobStateDeadline ||
		e == JobStateFailed ||
		e == JobStateCancelled ||
		e == JobStateNodeFail ||
		e == JobStateBootFail ||
		e == JobStatePending ||
		e == JobStateSuspended ||
		e == JobStateTimeout ||
		e == JobStatePreempted ||
		e == JobStateOutOfMemory
}
