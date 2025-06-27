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

func NewJobStartEvent(job *schema.Job) (CCMessage, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	return NewEvent("start_job", nil, nil, string(payload), time.Unix(job.StartTime, 0))
}

func NewJobStopEvent(job *schema.Job) (CCMessage, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	return NewEvent("stop_job", nil, nil, string(payload), time.Unix(job.StartTime, 0))
}

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
