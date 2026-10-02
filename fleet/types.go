// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import "strings"

// ServiceType is the short code a fleet member declares itself as. It is used
// as a database value, a directory name in the configuration tree and a NATS
// subject token.
type ServiceType string

const (
	ServiceMetricStore     ServiceType = "ccms" // cc-metric-store
	ServiceMetricCollector ServiceType = "ccmc" // cc-metric-collector
	ServiceBackend         ServiceType = "ccb"  // cc-backend
	ServiceEventStore      ServiceType = "cces" // cc-event-store
	ServiceSlurmAdapter    ServiceType = "ccsa" // cc-slurm-adapter
	ServiceNodeController  ServiceType = "ccnc" // cc-node-controller
	ServiceEnergyManager   ServiceType = "ccem" // cc-energy-manager
)

// ServiceTypes lists all known service types.
var ServiceTypes = []ServiceType{
	ServiceMetricStore,
	ServiceMetricCollector,
	ServiceBackend,
	ServiceEventStore,
	ServiceSlurmAdapter,
	ServiceNodeController,
	ServiceEnergyManager,
}

// Valid reports whether t is a known service type.
func (t ServiceType) Valid() bool {
	for _, known := range ServiceTypes {
		if t == known {
			return true
		}
	}
	return false
}

// Scope selects the registration endpoint and with it the configuration layers
// and the discovery bucket of a member.
type Scope string

const (
	// ScopeCluster is for services that belong to exactly one cluster.
	ScopeCluster Scope = "cluster"
	// ScopeInfra is for cluster-independent monitoring infrastructure.
	ScopeInfra Scope = "infra"
)

// RegisterRequest is the body of POST /api/fleet/register/{cluster|infra}/.
type RegisterRequest struct {
	Cluster     string            `json:"cluster,omitempty"`
	Hostname    string            `json:"hostname"`
	ServiceType ServiceType       `json:"serviceType"`
	MetaData    map[string]string `json:"metaData,omitempty"`
}

// RegisterResponse is the body of a successful (201) registration.
type RegisterResponse struct {
	InstanceID string `json:"instanceId"`
	// ConfigRevision is the revision this identity last acknowledged, or 0 if
	// it never pulled a configuration.
	ConfigRevision int64 `json:"configRevision"`
}

// Provider is one entry of a discovery roster.
type Provider struct {
	Type     ServiceType       `json:"type"`
	Hostname string            `json:"hostname"`
	State    string            `json:"state"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// Wire constants shared by client and server.
const (
	// HeartbeatMeasurement is the line-protocol measurement of a heartbeat.
	HeartbeatMeasurement = "fleet"
	// HeartbeatFunction is the value of the "function" tag of a heartbeat,
	// the only function cc-backend accepts on the heartbeat subject.
	HeartbeatFunction = "heartbeat"
	// DiscoveryMeasurement is the line-protocol measurement of a roster.
	DiscoveryMeasurement = "fleetdiscovery"
	// DefaultDiscoveryPrefix is the default NATS subject prefix of rosters.
	DefaultDiscoveryPrefix = "cc.fleet.discovery"
	// InfraBucket is the discovery bucket of infra-scope services.
	InfraBucket = "infra"
	// HeaderConfigRevision carries the config revision next to the ETag.
	HeaderConfigRevision = "X-CC-Config-Revision"
)

// safePathComponent mirrors cc-backend's registration check: hostname and
// cluster become directory names in the configuration tree.
func safePathComponent(s string) bool {
	return s != "" && s != "." && !strings.Contains(s, "..") && !strings.ContainsAny(s, `/\`)
}
