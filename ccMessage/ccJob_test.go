// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage

import (
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestNewJobStartEvent(t *testing.T) {
	job := &schema.Job{
		JobID:     12345,
		User:      "testuser",
		Project:   "testproject",
		Cluster:   "testcluster",
		StartTime: time.Now().Unix(),
	}

	msg, err := NewJobStartEvent(job)
	if err != nil {
		t.Fatalf("NewJobStartEvent failed: %v", err)
	}

	if msg.Name() != "start_job" {
		t.Errorf("Expected name 'start_job', got '%s'", msg.Name())
	}

	if !msg.IsEvent() {
		t.Error("Expected IsEvent() to return true")
	}

	eventName, ok := msg.IsJobEvent()
	if !ok {
		t.Error("Expected IsJobEvent() to return true")
	}
	if eventName != "start_job" {
		t.Errorf("Expected IsJobEvent() to return 'start_job', got '%s'", eventName)
	}
}

func TestNewJobStopEvent(t *testing.T) {
	job := &schema.Job{
		JobID:     12345,
		User:      "testuser",
		Project:   "testproject",
		Cluster:   "testcluster",
		StartTime: time.Now().Unix(),
	}

	msg, err := NewJobStopEvent(job)
	if err != nil {
		t.Fatalf("NewJobStopEvent failed: %v", err)
	}

	if msg.Name() != "stop_job" {
		t.Errorf("Expected name 'stop_job', got '%s'", msg.Name())
	}

	if !msg.IsEvent() {
		t.Error("Expected IsEvent() to return true")
	}

	eventName, ok := msg.IsJobEvent()
	if !ok {
		t.Error("Expected IsJobEvent() to return true")
	}
	if eventName != "stop_job" {
		t.Errorf("Expected IsJobEvent() to return 'stop_job', got '%s'", eventName)
	}
}

func TestGetJob_FromStartEvent(t *testing.T) {
	originalJob := &schema.Job{
		JobID:     12345,
		User:      "testuser",
		Project:   "testproject",
		Cluster:   "testcluster",
		StartTime: time.Now().Unix(),
	}

	msg, err := NewJobStartEvent(originalJob)
	if err != nil {
		t.Fatalf("NewJobStartEvent failed: %v", err)
	}

	retrievedJob, err := msg.GetJob()
	if err != nil {
		t.Fatalf("GetJob() failed: %v", err)
	}

	if retrievedJob.JobID != originalJob.JobID {
		t.Errorf("Expected JobID %d, got %d", originalJob.JobID, retrievedJob.JobID)
	}
	if retrievedJob.User != originalJob.User {
		t.Errorf("Expected User '%s', got '%s'", originalJob.User, retrievedJob.User)
	}
	if retrievedJob.Project != originalJob.Project {
		t.Errorf("Expected Project '%s', got '%s'", originalJob.Project, retrievedJob.Project)
	}
}

func TestIsJobEvent_NonJobEvent(t *testing.T) {
	msg, _ := NewEvent("other_event", nil, nil, "test", time.Now())

	_, ok := msg.IsJobEvent()
	if ok {
		t.Error("Expected IsJobEvent() to return false for non-job event")
	}
}

func TestIsJobEvent_NonEvent(t *testing.T) {
	msg, _ := NewMetric("test_metric", nil, nil, 1.0, time.Now())

	_, ok := msg.IsJobEvent()
	if ok {
		t.Error("Expected IsJobEvent() to return false for non-event message")
	}
}

func TestGetJob_InvalidJSON(t *testing.T) {
	msg, _ := NewEvent("start_job", nil, nil, "invalid json {", time.Now())

	_, err := msg.GetJob()
	if err == nil {
		t.Error("Expected GetJob() to fail with invalid JSON")
	}
}
