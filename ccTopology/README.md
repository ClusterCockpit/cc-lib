<!--
---
title: Topology package for ClusterCockpit
description: Topology package for ClusterCockpit
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccTopology/_index.md
---
-->


# ccTopology

The `ccTopology` package provides easy access to topology information, mainly for the local node.
It loads the topology once through the [`hwloc`](https://www-lb.open-mpi.org/projects/hwloc/)
library but stores it in own data structures for later access. Main purpose is to provide a
common interface for various [ClusterCockpit](https://clustercockpit.org/) components to
topology information.

**Note**: In order to use it, the environment variable `CGO_LDFLAGS="-L/path/to/lib/directory/of/hwloc` needs
to be set to the library directory of hwloc.

For transmission of the whole topology of a node, ccTopology's `Topology` type can be marshaled to JSON:

```go
func GetTopologyJSON() (json.RawMessage, error) {
    topo, err := ccTopology.LocalTopology()
    if err != nil {
        return nil, fmt.Errorf("Failed to init topology: %v", err.Error())
    }
    x, err := json.MarshalIndent(topo, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("Failed to marshal topology: %v", err.Error())
    }
    return x, nil
}
```

On the receiving side, it can be unmarshaled to ccTopology's `Topology` type:

```go
func GetNodeJSON(topologyJson json.RawMessage) (ccTopology.Topology, error) {
	topo, err := ccTopology.RemoteTopology(topologyJson)
	if err != nil {
		return nil, err
	}
    return topo, nil
}
```

There are additional helpers to get specific information out of node's topology:

```go
type Topology interface {
	GetHwthreads() []uint
	GetHwthreadStrings() []string
	GetSockets() []uint
	GetSocketStrings() []string
	GetDies() []uint
	GetDieStrings() []string
	GetCores() []uint
	GetCoreStrings() []string
	GetPciDevices() []uint
	GetPciDeviceStrings() []string
	GetHwthreadsOfSocket(socket uint) []uint
	GetHwthreadStringsOfSocket(socket uint) []string
	GetHwthreadsOfMemoryDomain(memoryDomain uint) []uint
	GetHwthreadStringsOfMemoryDomain(memoryDomain uint) []string
	GetNumaNodeOfPciDevice(address string) int
	CpuInfo() CpuInformation
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(in []byte) error
}
```

## Testing

```sh
$ make test
CGO_LDFLAGS="-L/modules/hwloc-2.4.0/lib" /modules/go-1.23.2/bin/go test
PASS
ok  	github.com/cc-lib/ccTopology	0.093s
cd test && CGO_LDFLAGS="-L/modules/hwloc-2.4.0/lib" /modules/go-1.23.2/bin/go test
PASS
ok  	github.com/cc-lib/ccTopology/test	0.014s

```