// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"fmt"
	"strconv"
)

// Accelerator represents a hardware accelerator (e.g., GPU, FPGA) attached to a compute node.
// Each accelerator has a unique identifier and type/model information.
type Accelerator struct {
	ID    string `json:"id"`    // Unique identifier for the accelerator (e.g., "0", "1", "GPU-0")
	Type  string `json:"type"`  // Type of accelerator (e.g., "Nvidia GPU", "AMD GPU")
	Model string `json:"model"` // Specific model name (e.g., "A100", "MI100")
}

// Topology defines the hardware topology of a compute node, mapping the hierarchical
// relationships between hardware threads, cores, sockets, memory domains, and accelerators.
//
// The topology is represented as nested arrays where indices represent hardware IDs:
//   - Node: Flat list of all hardware thread IDs on the node
//   - Socket: Hardware threads grouped by physical CPU socket
//   - Core: Hardware threads grouped by physical core
//   - MemoryDomain: Hardware threads grouped by NUMA domain
//   - Die: Optional grouping by CPU die within sockets
//   - Accelerators: List of attached hardware accelerators
type Topology struct {
	Node         []int          `json:"node"`                   // All hardware thread IDs on this node
	Socket       [][]int        `json:"socket"`                 // Hardware threads grouped by socket
	MemoryDomain [][]int        `json:"memoryDomain"`           // Hardware threads grouped by NUMA domain
	Die          [][]*int       `json:"die,omitempty"`          // Hardware threads grouped by die (optional)
	Core         [][]int        `json:"core"`                   // Hardware threads grouped by core
	Accelerators []*Accelerator `json:"accelerators,omitempty"` // Attached accelerators (GPUs, etc.)
}

// MetricValue represents a single metric measurement with its associated unit.
// Used for hardware performance characteristics like FLOP rates and memory bandwidth.
type MetricValue struct {
	Unit  Unit    `json:"unit"`  // Unit of measurement (e.g., FLOP/s, GB/s)
	Value float64 `json:"value"` // Numeric value of the measurement
}

// SubCluster represents a homogeneous partition of a cluster with identical hardware.
// A cluster may contain multiple subclusters with different processor types or configurations.
type SubCluster struct {
	Name            string          `json:"name"`                      // Name of the subcluster (e.g., "main", "gpu", "bigmem")
	Nodes           string          `json:"nodes"`                     // Node list in condensed format (e.g., "node[001-100]")
	ProcessorType   string          `json:"processorType"`             // CPU model (e.g., "Intel Xeon Gold 6148")
	Topology        Topology        `json:"topology"`                  // Hardware topology of nodes in this subcluster
	FlopRateScalar  MetricValue     `json:"flopRateScalar"`            // Theoretical scalar FLOP rate per node
	FlopRateSimd    MetricValue     `json:"flopRateSimd"`              // Theoretical SIMD FLOP rate per node
	MemoryBandwidth MetricValue     `json:"memoryBandwidth"`           // Theoretical memory bandwidth per node
	MetricConfig    []*MetricConfig `json:"metricConfig,omitempty"`    // Subcluster-specific metric configurations
	Footprint       []string        `json:"footprint,omitempty"`       // Default footprint metrics for jobs
	EnergyFootprint []string        `json:"energyFootprint,omitempty"` // Energy-related footprint metrics
	SocketsPerNode  int             `json:"socketsPerNode"`            // Number of CPU sockets per node
	CoresPerSocket  int             `json:"coresPerSocket"`            // Number of cores per CPU socket
	ThreadsPerCore  int             `json:"threadsPerCore"`            // Number of hardware threads per core (SMT level)
}

// Metric defines thresholds for a performance metric used in job classification and alerts.
// Thresholds help categorize job performance: peak (excellent), normal (good), caution (concerning), alert (problem).
type Metric struct {
	Name    string  `json:"name"`    // Metric name (e.g., "cpu_load", "mem_used")
	Unit    Unit    `json:"unit"`    // Unit of measurement
	Peak    float64 `json:"peak"`    // Peak/maximum expected value (best performance)
	Normal  float64 `json:"normal"`  // Normal/typical value (good performance)
	Caution float64 `json:"caution"` // Caution threshold (concerning but not critical)
	Alert   float64 `json:"alert"`   // Alert threshold (requires attention)
}

// SubClusterConfig extends Metric with subcluster-specific metric configuration.
// Allows overriding metric settings for specific subclusters within a cluster.
type SubClusterConfig struct {
	Metric               // Embedded metric thresholds
	Footprint     string `json:"footprint,omitempty"` // Footprint category for this metric
	Energy        string `json:"energy"`              // Energy measurement configuration
	LowerIsBetter bool   `json:"lowerIsBetter"`       // Whether lower values indicate better performance
	Restrict      bool   `json:"restrict"`            // Restrict visibility to non user roles
	Remove        bool   `json:"remove"`              // Whether to exclude this metric for this subcluster
}

// MetricConfig defines the configuration for a performance metric at the cluster level.
// Specifies how the metric is collected, aggregated, and evaluated across the cluster.
type MetricConfig struct {
	Metric                            // Embedded metric thresholds
	Energy        string              `json:"energy"`                // Energy measurement method
	Scope         MetricScope         `json:"scope"`                 // Metric scope (node, socket, core, etc.)
	Aggregation   string              `json:"aggregation"`           // Aggregation function (avg, sum, min, max)
	Footprint     string              `json:"footprint,omitempty"`   // Footprint category
	SubClusters   []*SubClusterConfig `json:"subClusters,omitempty"` // Subcluster-specific overrides
	Timestep      int                 `json:"timestep"`              // Measurement interval in seconds
	Restrict      bool                `json:"restrict"`              // Restrict visibility to non user roles
	LowerIsBetter bool                `json:"lowerIsBetter"`         // Whether lower values are better
}

// Cluster represents a complete HPC cluster configuration.
// A cluster consists of one or more subclusters and defines metric collection/evaluation settings.
type Cluster struct {
	Name         string          `json:"name"`         // Unique cluster name (e.g., "fritz", "alex")
	MetricConfig []*MetricConfig `json:"metricConfig"` // Cluster-wide metric configurations
	SubClusters  []*SubCluster   `json:"subClusters"`  // Homogeneous partitions within the cluster
}

// ClusterSupport indicates which subclusters within a cluster support a particular metric.
// Used to track metric availability across heterogeneous clusters.
type ClusterSupport struct {
	Cluster     string   `json:"cluster"`     // Cluster name
	SubClusters []string `json:"subclusters"` // List of subcluster names supporting this metric
}

// GlobalMetricListItem represents a metric in the global metric catalog.
// Tracks which clusters and subclusters support this metric across the entire system.
type GlobalMetricListItem struct {
	Name         string      `json:"name"`                // Metric name
	Unit         Unit        `json:"unit"`                // Unit of measurement
	Scope        MetricScope `json:"scope"`               // Metric scope level
	Footprint    string      `json:"footprint,omitempty"` // Footprint category
	Restrict     bool
	Availability []ClusterSupport `json:"availability"` // Where this metric is available
}

// GetSocketsFromHWThreads returns socket IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned sockets
// are present in the input list (i.e., the job has exclusive access to those sockets).
func (topo *Topology) GetSocketsFromHWThreads(
	hwthreads []int,
) (sockets []int, exclusive bool) {
	// Build hwthread -> socket lookup map
	hwthreadToSocket := make(map[int]int, len(topo.Node))
	for socket, hwthreadsInSocket := range topo.Socket {
		for _, hwt := range hwthreadsInSocket {
			hwthreadToSocket[hwt] = socket
		}
	}
	// Count hwthreads per socket from input
	socketsMap := make(map[int]int)
	for _, hwt := range hwthreads {
		if socket, ok := hwthreadToSocket[hwt]; ok {
			socketsMap[socket]++
		}
	}
	// Build result and check exclusivity
	exclusive = true
	hwthreadsPerSocket := len(topo.Node) / len(topo.Socket)
	sockets = make([]int, 0, len(socketsMap))
	for socket, count := range socketsMap {
		sockets = append(sockets, socket)
		exclusive = exclusive && count == hwthreadsPerSocket
	}
	return sockets, exclusive
}

// GetSocketsFromCores returns socket IDs that contain any of the given cores.
// The exclusive return value is true if all hardware threads in the returned sockets
// belong to cores in the input list (i.e., the job has exclusive access to those sockets).
func (topo *Topology) GetSocketsFromCores(
	cores []int,
) (sockets []int, exclusive bool) {
	// Build hwthread -> socket lookup map
	hwthreadToSocket := make(map[int]int, len(topo.Node))
	for socket, hwthreadsInSocket := range topo.Socket {
		for _, hwt := range hwthreadsInSocket {
			hwthreadToSocket[hwt] = socket
		}
	}
	// Count hwthreads per socket from input cores
	socketsMap := make(map[int]int)
	for _, core := range cores {
		for _, hwt := range topo.Core[core] {
			if socket, ok := hwthreadToSocket[hwt]; ok {
				socketsMap[socket]++
			}
		}
	}
	// Build result and check exclusivity
	exclusive = true
	hwthreadsPerSocket := len(topo.Node) / len(topo.Socket)
	sockets = make([]int, 0, len(socketsMap))
	for socket, count := range socketsMap {
		sockets = append(sockets, socket)
		exclusive = exclusive && count == hwthreadsPerSocket
	}
	return sockets, exclusive
}

// GetCoresFromHWThreads returns core IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned cores
// are present in the input list (i.e., the job has exclusive access to those cores).
func (topo *Topology) GetCoresFromHWThreads(
	hwthreads []int,
) (cores []int, exclusive bool) {
	// Build hwthread -> core lookup map
	hwthreadToCore := make(map[int]int, len(topo.Node))
	for core, hwthreadsInCore := range topo.Core {
		for _, hwt := range hwthreadsInCore {
			hwthreadToCore[hwt] = core
		}
	}
	// Count hwthreads per core from input
	coresMap := make(map[int]int)
	for _, hwt := range hwthreads {
		if core, ok := hwthreadToCore[hwt]; ok {
			coresMap[core]++
		}
	}
	// Build result and check exclusivity
	exclusive = true
	hwthreadsPerCore := len(topo.Node) / len(topo.Core)
	cores = make([]int, 0, len(coresMap))
	for core, count := range coresMap {
		cores = append(cores, core)
		exclusive = exclusive && count == hwthreadsPerCore
	}
	return cores, exclusive
}

// GetMemoryDomainsFromHWThreads returns memory domain IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned memory domains
// are present in the input list (i.e., the job has exclusive access to those memory domains).
func (topo *Topology) GetMemoryDomainsFromHWThreads(
	hwthreads []int,
) (memDoms []int, exclusive bool) {
	// Build hwthread -> memory domain lookup map
	hwthreadToMemDom := make(map[int]int, len(topo.Node))
	for memDom, hwthreadsInMemDom := range topo.MemoryDomain {
		for _, hwt := range hwthreadsInMemDom {
			hwthreadToMemDom[hwt] = memDom
		}
	}
	// Count hwthreads per memory domain from input
	memDomsMap := make(map[int]int)
	for _, hwt := range hwthreads {
		if memDom, ok := hwthreadToMemDom[hwt]; ok {
			memDomsMap[memDom]++
		}
	}
	// Build result and check exclusivity
	exclusive = true
	hwthreadsPerMemDom := len(topo.Node) / len(topo.MemoryDomain)
	memDoms = make([]int, 0, len(memDomsMap))
	for memDom, count := range memDomsMap {
		memDoms = append(memDoms, memDom)
		exclusive = exclusive && count == hwthreadsPerMemDom
	}
	return memDoms, exclusive
}

// GetAcceleratorID converts an integer accelerator index to its string ID.
// Returns an error if the index is out of range.
func (topo *Topology) GetAcceleratorID(id int) (string, error) {
	if id < 0 {
		return "", fmt.Errorf("accelerator index %d is negative", id)
	}
	if id >= len(topo.Accelerators) {
		return "", fmt.Errorf("accelerator index %d out of range (have %d accelerators)", id, len(topo.Accelerators))
	}
	return topo.Accelerators[id].ID, nil
}

// GetAcceleratorIDs returns a list of all accelerator IDs as strings.
func (topo *Topology) GetAcceleratorIDs() []string {
	accels := make([]string, 0, len(topo.Accelerators))
	for _, accel := range topo.Accelerators {
		accels = append(accels, accel.ID)
	}
	return accels
}

// GetAcceleratorIDsAsInt attempts to convert all accelerator IDs to integers.
// Returns an error if any accelerator ID is not a valid integer.
// This method assumes accelerator IDs are numeric strings.
func (topo *Topology) GetAcceleratorIDsAsInt() ([]int, error) {
	accels := make([]int, 0, len(topo.Accelerators))
	for _, accel := range topo.Accelerators {
		id, err := strconv.Atoi(accel.ID)
		if err != nil {
			return nil, fmt.Errorf("accelerator ID %q is not a valid integer: %w", accel.ID, err)
		}
		accels = append(accels, id)
	}
	return accels, nil
}
