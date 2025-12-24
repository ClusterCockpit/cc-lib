// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccmessage_test

import (
	"fmt"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

func ExampleNewMetric() {
	msg, err := ccmessage.NewMetric(
		"cpu_usage",
		map[string]string{"hostname": "node001", "type": "node"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Unix(1234567890, 0),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Name: %s\n", msg.Name())
	if value, ok := msg.GetMetricValue(); ok {
		fmt.Printf("Value: %.1f\n", value)
	}
	// Output:
	// Name: cpu_usage
	// Value: 75.5
}

func ExampleNewEvent() {
	msg, err := ccmessage.NewEvent(
		"node_down",
		map[string]string{"severity": "critical"},
		nil,
		"Node node001 is unreachable",
		time.Unix(1234567890, 0),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Event: %s\n", msg.Name())
	if event, ok := msg.GetEventValue(); ok {
		fmt.Printf("Details: %s\n", event)
	}
	// Output:
	// Event: node_down
	// Details: Node node001 is unreachable
}

func ExampleCCMessage_MessageType() {
	metric, _ := ccmessage.NewMetric("test", nil, nil, 123, time.Now())
	event, _ := ccmessage.NewEvent("test", nil, nil, "event", time.Now())
	log, _ := ccmessage.NewLog("test", nil, nil, "log", time.Now())

	fmt.Printf("Metric type: %s\n", metric.MessageType())
	fmt.Printf("Event type: %s\n", event.MessageType())
	fmt.Printf("Log type: %s\n", log.MessageType())
	// Output:
	// Metric type: metric
	// Event type: event
	// Log type: log
}

func ExampleCCMessage_ToLineProtocol() {
	msg, _ := ccmessage.NewMetric(
		"cpu_usage",
		map[string]string{"hostname": "node001"},
		map[string]string{"unit": "percent"},
		75.5,
		time.Unix(1234567890, 0),
	)

	lp := msg.ToLineProtocol(map[string]bool{})
	fmt.Println(lp)
	// Output:
	// cpu_usage,hostname=node001 value=75.5 1234567890000000000
}

func ExampleCCMessage_ToLineProtocol_withMeta() {
	msg, _ := ccmessage.NewMetric(
		"memory_used",
		map[string]string{"hostname": "node001"},
		map[string]string{"unit": "bytes"},
		1024,
		time.Unix(1234567890, 0),
	)

	lp := msg.ToLineProtocol(map[string]bool{"unit": true})
	fmt.Println(lp)
	// Output:
	// memory_used,hostname=node001,unit=bytes value=1024i 1234567890000000000
}

func ExampleFromMessage() {
	original, _ := ccmessage.NewMetric("cpu_usage", nil, nil, 50.0, time.Now())

	copy := ccmessage.FromMessage(original)
	copy.AddTag("modified", "true")

	fmt.Printf("Original has tag: %v\n", original.HasTag("modified"))
	fmt.Printf("Copy has tag: %v\n", copy.HasTag("modified"))
	// Output:
	// Original has tag: false
	// Copy has tag: true
}

func ExampleFromBytes() {
	data := []byte("cpu_usage,hostname=node001 value=75.5 1234567890000000000")

	messages, err := ccmessage.FromBytes(data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, msg := range messages {
		fmt.Printf("Metric: %s\n", msg.Name())
		if value, ok := msg.GetMetricValue(); ok {
			fmt.Printf("Value: %v\n", value)
		}
	}
	// Output:
	// Metric: cpu_usage
	// Value: 75.5
}

func ExampleCCMessage_IsMetric() {
	metric, _ := ccmessage.NewMetric("test", nil, nil, 123, time.Now())
	event, _ := ccmessage.NewEvent("test", nil, nil, "event", time.Now())

	fmt.Printf("Metric is metric: %v\n", metric.IsMetric())
	fmt.Printf("Event is metric: %v\n", event.IsMetric())
	// Output:
	// Metric is metric: true
	// Event is metric: false
}

func ExampleNewGetControl() {
	msg, _ := ccmessage.NewGetControl(
		"sampling_rate",
		map[string]string{"component": "collector"},
		nil,
		time.Unix(1234567890, 0),
	)

	if method, ok := msg.GetControlMethod(); ok {
		fmt.Printf("Method: %s\n", method)
	}
	fmt.Printf("Parameter: %s\n", msg.Name())
	// Output:
	// Method: GET
	// Parameter: sampling_rate
}

func ExampleNewPutControl() {
	msg, _ := ccmessage.NewPutControl(
		"sampling_rate",
		nil,
		nil,
		"10",
		time.Unix(1234567890, 0),
	)

	if method, ok := msg.GetControlMethod(); ok {
		fmt.Printf("Method: %s\n", method)
	}
	if value, ok := msg.GetControlValue(); ok {
		fmt.Printf("New value: %s\n", value)
	}
	// Output:
	// Method: PUT
	// New value: 10
}

func ExampleNewMessage_validation() {
	_, err1 := ccmessage.NewMessage("", nil, nil, map[string]any{"value": 1}, time.Now())
	_, err2 := ccmessage.NewMessage("test", nil, nil, map[string]any{"value": 1}, time.Time{})
	_, err3 := ccmessage.NewMessage("test", nil, nil, map[string]any{}, time.Now())

	fmt.Printf("Empty name error: %v\n", err1 != nil)
	fmt.Printf("Zero timestamp error: %v\n", err2 != nil)
	fmt.Printf("No fields error: %v\n", err3 != nil)
	// Output:
	// Empty name error: true
	// Zero timestamp error: true
	// No fields error: true
}
