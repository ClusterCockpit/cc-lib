// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccTopology

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	_, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}
}

func TestGetHwThreads(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	hwts := topo.GetHwthreads()
	if len(hwts) == 0 {
		t.Error("no hwthreads reported for system")
	}
	Shwts := topo.GetHwthreadStrings()
	if len(Shwts) == 0 {
		t.Error("no hwthreads reported for system")
	}
	t.Log("[", strings.Join(Shwts, ","), "]")
}

func TestGetSockets(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	socks := topo.GetSockets()
	if len(socks) == 0 {
		t.Error("no sockets reported for system")
	}
	Ssocks := topo.GetSocketStrings()
	if len(Ssocks) == 0 {
		t.Error("no sockets reported for system")
	}
	t.Log("[", strings.Join(Ssocks, ","), "]")
}

func TestGetDies(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	socks := topo.GetDies()
	if len(socks) == 0 {
		t.Error("no dies reported for system")
	}
	Ssocks := topo.GetDieStrings()
	if len(Ssocks) == 0 {
		t.Error("no dies reported for system")
	}
	t.Log("[", strings.Join(Ssocks, ","), "]")
}

func TestGetCores(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	socks := topo.GetCores()
	if len(socks) == 0 {
		t.Error("no cores reported for system")
	}
	Ssocks := topo.GetCoreStrings()
	if len(Ssocks) == 0 {
		t.Error("no cores reported for system")
	}
	t.Log("[", strings.Join(Ssocks, ","), "]")
}

func TestGetMemoryDomains(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	doms := topo.GetMemoryDomains()
	if len(doms) == 0 {
		t.Error("no memory domains reported for system")
	}
	Sdoms := topo.GetMemoryDomainStrings()
	if len(Sdoms) == 0 {
		t.Error("no memory domains reported for system")
	}
	t.Log("[", strings.Join(Sdoms, ","), "]")
}

func TestGetPciDevices(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}
	socks := topo.GetPciDevices()
	if len(socks) == 0 {
		t.Error("no pci devices reported for system")
	}
	Ssocks := topo.GetPciDeviceStrings()
	if len(Ssocks) == 0 {
		t.Error("no pci devices reported for system")
	}
	t.Log("[", strings.Join(Ssocks, ","), "]")
}

func TestGetHwthreadsOfSocket(t *testing.T) {
	var topo Topology
	socket := uint(0)

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	hwts := topo.GetHwthreadsOfSocket(socket)
	if len(hwts) == 0 {
		t.Error("no hwthreads reported for socket ", socket)
	}
	Shwts := topo.GetHwthreadStringsOfSocket(socket)
	if len(Shwts) == 0 {
		t.Error("no hwthreads reported for socket ", socket)
	}
	t.Log("[", strings.Join(Shwts, ","), "]")
}

func TestGetHwthreadsOfMemoryDomain(t *testing.T) {
	var topo Topology
	memoryDomain := uint(0)

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}
	hwts := topo.GetHwthreadsOfMemoryDomain(memoryDomain)
	if len(hwts) == 0 {
		t.Error("no hwthreads reported for memory domain ", memoryDomain)
	}
	Shwts := topo.GetHwthreadStringsOfMemoryDomain(memoryDomain)
	if len(Shwts) == 0 {
		t.Error("no hwthreads reported for memory domai ", memoryDomain)
	}
	t.Log("[", strings.Join(Shwts, ","), "]")
}

func TestCpuinfo(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}

	cpuinfo := topo.CpuInfo()

	if cpuinfo.NumHWthreads == 0 {
		t.Error("failed to detect number of hwthreads")
	}
	if cpuinfo.NumCores == 0 {
		t.Error("failed to detect number of cores")
	}
	if cpuinfo.NumSockets == 0 {
		t.Error("failed to detect number of sockets")
	}
	if cpuinfo.NumDies == 0 {
		t.Error("failed to detect number of dies")
	}
	if cpuinfo.NumNumaDomains == 0 {
		t.Error("failed to detect number of NUMA domains")
	}
}

func TestRemoteTopology(t *testing.T) {
	var topo Topology

	topo, err := LocalTopology()
	if err != nil {
		t.Error("failed to initialize topology: ", err.Error())
	}
	topoJson, err := json.Marshal(topo)
	if err != nil {
		t.Error("failed to get JSON of topology: ", err.Error())
	}

	rtopo, err := RemoteTopology(topoJson)
	if err != nil {
		t.Error("failed to parse JSON of remote topology: ", err.Error())
	}
	cpuinfo := rtopo.CpuInfo()

	if cpuinfo.NumHWthreads == 0 {
		t.Error("failed to detect number of hwthreads")
	}
	if cpuinfo.NumCores == 0 {
		t.Error("failed to detect number of cores")
	}
	if cpuinfo.NumSockets == 0 {
		t.Error("failed to detect number of sockets")
	}
	if cpuinfo.NumDies == 0 {
		t.Error("failed to detect number of dies")
	}
	if cpuinfo.NumNumaDomains == 0 {
		t.Error("failed to detect number of NUMA domains")
	}
}
