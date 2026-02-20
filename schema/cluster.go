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

	// Cache maps for faster lookups
	hwthreadToSocket       map[int][]int
	hwthreadToCore         map[int][]int
	hwthreadToMemoryDomain map[int][]int
	coreToSocket           map[int][]int
	memoryDomainToSocket   map[int]int // New: Direct mapping from memory domain to socket
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
	Name         string           `json:"name"`                // Metric name
	Unit         Unit             `json:"unit"`                // Unit of measurement
	Scope        MetricScope      `json:"scope"`               // Metric scope level
	Footprint    string           `json:"footprint,omitempty"` // Footprint category
	Availability []ClusterSupport `json:"availability"`        // Where this metric is available
}

// InitTopologyMaps initializes the topology mapping caches
func (topo *Topology) InitTopologyMaps() {
	// Initialize maps
	topo.hwthreadToSocket = make(map[int][]int)
	topo.hwthreadToCore = make(map[int][]int)
	topo.hwthreadToMemoryDomain = make(map[int][]int)
	topo.coreToSocket = make(map[int][]int)
	topo.memoryDomainToSocket = make(map[int]int)

	// Build hwthread to socket mapping
	for socketID, hwthreads := range topo.Socket {
		for _, hwthread := range hwthreads {
			topo.hwthreadToSocket[hwthread] = append(topo.hwthreadToSocket[hwthread], socketID)
		}
	}

	// Build hwthread to core mapping
	for coreID, hwthreads := range topo.Core {
		for _, hwthread := range hwthreads {
			topo.hwthreadToCore[hwthread] = append(topo.hwthreadToCore[hwthread], coreID)
		}
	}

	// Build hwthread to memory domain mapping
	for memDomID, hwthreads := range topo.MemoryDomain {
		for _, hwthread := range hwthreads {
			topo.hwthreadToMemoryDomain[hwthread] = append(topo.hwthreadToMemoryDomain[hwthread], memDomID)
		}
	}

	// Build core to socket mapping
	for coreID, hwthreads := range topo.Core {
		socketSet := make(map[int]struct{})
		for _, hwthread := range hwthreads {
			for socketID := range topo.hwthreadToSocket[hwthread] {
				socketSet[socketID] = struct{}{}
			}
		}
		topo.coreToSocket[coreID] = make([]int, 0, len(socketSet))
		for socketID := range socketSet {
			topo.coreToSocket[coreID] = append(topo.coreToSocket[coreID], socketID)
		}
	}

	// Build memory domain to socket mapping
	for memDomID, hwthreads := range topo.MemoryDomain {
		if len(hwthreads) > 0 {
			// Use the first hwthread to determine the socket
			if socketIDs, ok := topo.hwthreadToSocket[hwthreads[0]]; ok && len(socketIDs) > 0 {
				topo.memoryDomainToSocket[memDomID] = socketIDs[0]
			}
		}
	}
}

// EnsureTopologyMaps ensures that the topology maps are initialized
func (topo *Topology) EnsureTopologyMaps() {
	if topo.hwthreadToSocket == nil {
		topo.InitTopologyMaps()
	}
}

// GetSocketsFromHWThreads returns socket IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned sockets
// are present in the input list (i.e., the job has exclusive access to those sockets).
func (topo *Topology) GetSocketsFromHWThreads(
	hwthreads []int,
) (sockets []int, exclusive bool) {
	// Ensure Init -> contains memory domain lookup map
	topo.EnsureTopologyMaps()

	// Count hwthreads per socket from input
	socketsMap := make(map[int]int)
	for _, hwt := range hwthreads {
		for _, socketID := range topo.hwthreadToSocket[hwt] {
			socketsMap[socketID]++
		}
	}

	// Build result and check exclusivity
	exclusive = true
	sockets = make([]int, 0, len(socketsMap))
	for socket, count := range socketsMap {
		sockets = append(sockets, socket)
		// Check if all hwthreads in this socket are in our input list
		exclusive = exclusive && count == len(topo.Socket[socket])
	}
	return sockets, exclusive
}

// GetSocketsFromCores returns socket IDs that contain any of the given cores.
// The exclusive return value is true if all hardware threads in the returned sockets
// belong to cores in the input list (i.e., the job has exclusive access to those sockets).
func (topo *Topology) GetSocketsFromCores(
	cores []int,
) (sockets []int, exclusive bool) {
	// Ensure Init -> contains memory domain lookup map
	topo.EnsureTopologyMaps()

	socketsMap := make(map[int]int)
	for _, core := range cores {
		for _, socketID := range topo.coreToSocket[core] {
			socketsMap[socketID]++
		}
	}

	// Build result and check exclusivity
	exclusive = true
	sockets = make([]int, 0, len(socketsMap))
	for socket, count := range socketsMap {
		sockets = append(sockets, socket)
		// Count total cores in this socket
		totalCoresInSocket := 0
		for _, hwthreads := range topo.Core {
			for _, hwthread := range hwthreads {
				for _, sID := range topo.hwthreadToSocket[hwthread] {
					if sID == socket {
						totalCoresInSocket++
						break
					}
				}
			}
		}
		exclusive = exclusive && count == totalCoresInSocket
	}
	return sockets, exclusive
}

// GetCoresFromHWThreads returns core IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned cores
// are present in the input list (i.e., the job has exclusive access to those cores).
func (topo *Topology) GetCoresFromHWThreads(
	hwthreads []int,
) (cores []int, exclusive bool) {
	// Ensure Init -> contains memory domain lookup map
	topo.EnsureTopologyMaps()

	coresMap := make(map[int]int)
	for _, hwt := range hwthreads {
		for _, coreID := range topo.hwthreadToCore[hwt] {
			coresMap[coreID]++
		}
	}

	// Build result and check exclusivity
	exclusive = true
	cores = make([]int, 0, len(coresMap))
	for core, count := range coresMap {
		cores = append(cores, core)
		// Check if all hwthreads in this core are in our input list
		exclusive = exclusive && count == len(topo.Core[core])
	}
	return cores, exclusive
}

// GetMemoryDomainsFromHWThreads returns memory domain IDs that contain any of the given hardware threads.
// The exclusive return value is true if all hardware threads in the returned memory domains
// are present in the input list (i.e., the job has exclusive access to those memory domains).
func (topo *Topology) GetMemoryDomainsFromHWThreads(
	hwthreads []int,
) (memDoms []int, exclusive bool) {
	// Ensure Init -> contains memory domain lookup map
	topo.EnsureTopologyMaps()

	memDomsMap := make(map[int]int)
	for _, hwt := range hwthreads {
		for _, memDomID := range topo.hwthreadToMemoryDomain[hwt] {
			memDomsMap[memDomID]++
		}
	}

	// Build result and check exclusivity
	exclusive = true
	memDoms = make([]int, 0, len(memDomsMap))
	for memDom, count := range memDomsMap {
		memDoms = append(memDoms, memDom)
		// Check if all hwthreads in this memory domain are in our input list
		exclusive = exclusive && count == len(topo.MemoryDomain[memDom])
	}
	return memDoms, exclusive
}

// GetMemoryDomainsBySocket can now use the direct mapping
func (topo *Topology) GetMemoryDomainsBySocket(domainIDs []int) (map[int][]int, error) {
	socketToDomains := make(map[int][]int)
	for _, domainID := range domainIDs {
		if domainID < 0 || domainID >= len(topo.MemoryDomain) || len(topo.MemoryDomain[domainID]) == 0 {
			return nil, fmt.Errorf("MemoryDomain %d is invalid or empty", domainID)
		}

		socketID, ok := topo.memoryDomainToSocket[domainID]
		if !ok {
			return nil, fmt.Errorf("MemoryDomain %d could not be assigned to any socket", domainID)
		}

		socketToDomains[socketID] = append(socketToDomains[socketID], domainID)
	}

	return socketToDomains, nil
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
// Capacity is pre-allocated to improve efficiency.
func (topo *Topology) GetAcceleratorIDs() []string {
	if len(topo.Accelerators) == 0 {
		return []string{}
	}

	accels := make([]string, 0, len(topo.Accelerators))
	for _, accel := range topo.Accelerators {
		accels = append(accels, accel.ID)
	}
	return accels
}

// GetAcceleratorIDsAsInt attempts to convert all accelerator IDs to integers.
// Returns an error if any accelerator ID is not a valid integer.
// This method assumes accelerator IDs are numeric strings.
// Capacity is pre-allocated to improve efficiency.
func (topo *Topology) GetAcceleratorIDsAsInt() ([]int, error) {
	if len(topo.Accelerators) == 0 {
		return []int{}, nil
	}

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
