// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

// heartbeatPayload is the JSON object in the "event" field of a heartbeat.
type heartbeatPayload struct {
	InstanceID string `json:"instanceId"`
}

// Bucket returns the discovery bucket of a member: its cluster for a
// cluster-scope service, or InfraBucket when cluster is empty.
func Bucket(cluster string) string {
	if cluster == "" {
		return InfraBucket
	}
	return cluster
}

// DiscoverySubject returns the NATS subject on which a consumer of the given
// type in the given bucket receives its roster. An empty prefix selects
// DefaultDiscoveryPrefix.
func DiscoverySubject(prefix, bucket string, consumer ServiceType) string {
	if prefix == "" {
		prefix = DefaultDiscoveryPrefix
	}
	return prefix + "." + bucket + "." + string(consumer)
}

// EncodeHeartbeat encodes one heartbeat line per instance id. Several lines in
// one message are how an edge aggregator batches heartbeats. cc-backend
// ignores the timestamp, but the line protocol requires one.
func EncodeHeartbeat(t time.Time, instanceIDs ...string) ([]byte, error) {
	if len(instanceIDs) == 0 {
		return nil, errors.New("fleet: heartbeat without instance id")
	}

	var buf bytes.Buffer
	for _, id := range instanceIDs {
		if id == "" {
			return nil, errors.New("fleet: heartbeat with empty instance id")
		}
		payload, err := json.Marshal(heartbeatPayload{InstanceID: id})
		if err != nil {
			return nil, err
		}
		msg, err := lp.NewEvent(HeartbeatMeasurement,
			map[string]string{"function": HeartbeatFunction}, nil, string(payload), t)
		if err != nil {
			return nil, err
		}
		writeLine(&buf, msg.ToLineProtocol(nil))
	}
	return buf.Bytes(), nil
}

// ParseHeartbeat validates one decoded message as a heartbeat and returns its
// instance id. Unknown JSON keys in the payload are rejected.
func ParseHeartbeat(m lp.CCMessage) (string, error) {
	if m.Name() != HeartbeatMeasurement {
		return "", fmt.Errorf("fleet: unexpected measurement %q", m.Name())
	}
	if function, _ := m.GetTag("function"); function != HeartbeatFunction {
		return "", fmt.Errorf("fleet: unexpected function %q", function)
	}
	payload, ok := m.GetEventValue()
	if !ok {
		return "", errors.New("fleet: heartbeat is missing the event field")
	}

	var hb heartbeatPayload
	dec := json.NewDecoder(strings.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&hb); err != nil {
		return "", fmt.Errorf("fleet: decoding heartbeat payload: %w", err)
	}
	if hb.InstanceID == "" {
		return "", errors.New("fleet: heartbeat without instance id")
	}
	return hb.InstanceID, nil
}

// DecodeHeartbeats decodes a heartbeat message, which may hold several lines,
// and returns the instance ids. It fails on the first invalid line.
func DecodeHeartbeats(data []byte) ([]string, error) {
	msgs, err := lp.FromBytes(data)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		id, err := ParseHeartbeat(m)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// EncodeRoster encodes the roster for one (bucket, consumer type) subject.
func EncodeRoster(bucket string, consumer ServiceType, providers []Provider, t time.Time) ([]byte, error) {
	if providers == nil {
		providers = []Provider{}
	}
	payload, err := json.Marshal(providers)
	if err != nil {
		return nil, err
	}
	tags := map[string]string{"cluster": bucket, "type": string(consumer)}
	msg, err := lp.NewEvent(DiscoveryMeasurement, tags, nil, string(payload), t)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writeLine(&buf, msg.ToLineProtocol(nil))
	return buf.Bytes(), nil
}

// DecodeRoster decodes a roster message into the full provider list. An empty
// roster yields an empty, non-nil slice.
func DecodeRoster(data []byte) ([]Provider, error) {
	msgs, err := lp.FromBytes(data)
	if err != nil {
		return nil, err
	}
	if len(msgs) != 1 {
		return nil, fmt.Errorf("fleet: roster message holds %d lines, want 1", len(msgs))
	}
	m := msgs[0]
	if m.Name() != DiscoveryMeasurement {
		return nil, fmt.Errorf("fleet: unexpected measurement %q", m.Name())
	}
	payload, ok := m.GetEventValue()
	if !ok {
		return nil, errors.New("fleet: roster is missing the event field")
	}

	providers := []Provider{}
	if err := json.Unmarshal([]byte(payload), &providers); err != nil {
		return nil, fmt.Errorf("fleet: decoding roster payload: %w", err)
	}
	return providers, nil
}

func writeLine(buf *bytes.Buffer, line string) {
	buf.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		buf.WriteByte('\n')
	}
}
