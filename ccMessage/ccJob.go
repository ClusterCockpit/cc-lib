// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccmessage

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ClusterCockpit/cc-lib/schema"
)

// NewJobStartEvent creates an event message for a job start.
// The job information is serialized to JSON and embedded in the event payload.
//
// Parameters:
//   - job: Pointer to the schema.Job structure containing job information
//
// Returns a CCMessage with name "start_job" and the job data serialized as JSON in the event field.
// The timestamp is set to the job's start time.
func NewJobStartEvent(job *schema.Job) (CCMessage, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	return NewEvent("start_job", nil, nil, string(payload), time.Unix(job.StartTime, 0))
}

// NewJobStopEvent creates an event message for a job stop.
// The job information is serialized to JSON and embedded in the event payload.
//
// Parameters:
//   - job: Pointer to the schema.Job structure containing job information
//
// Returns a CCMessage with name "stop_job" and the job data serialized as JSON in the event field.
// The timestamp is set to the job's start time.
func NewJobStopEvent(job *schema.Job) (CCMessage, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	return NewEvent("stop_job", nil, nil, string(payload), time.Unix(job.StartTime, 0))
}

// IsJobEvent checks if the message is a job-related event (start_job or stop_job).
//
// Returns:
//   - name: The event name ("start_job" or "stop_job") if the message is a job event
//   - ok: true if the message is a job event, false otherwise
func (m *ccMessage) IsJobEvent() (string, bool) {
	if !m.IsEvent() {
		return "", false
	}

	name := m.name

	if name == "start_job" || name == "stop_job" {
		return name, true
	}

	return "", false
}

// GetJob deserializes the job information from a job event message.
// The event payload is expected to contain a JSON-serialized schema.Job structure.
//
// Returns:
//   - job: Pointer to the deserialized schema.Job structure
//   - err: Error if deserialization fails or if unknown fields are present
func (m *ccMessage) GetJob() (job *schema.Job, err error) {
	value := m.GetEventValue()
	d := json.NewDecoder(strings.NewReader(value))
	d.DisallowUnknownFields()
	job = &schema.Job{}

	if err = d.Decode(job); err == nil {
		return job, nil
	} else {
		return nil, err
	}
}
