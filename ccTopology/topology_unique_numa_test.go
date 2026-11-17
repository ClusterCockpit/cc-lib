// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"reflect"
	"testing"
)

func TestTopologyNUMAUnique(t *testing.T) {
	topo := topologyRegressionLocal(t, "package:2 numa:1 core:2 pu:2")
	if got := topo.GetMemoryDomains(); !reflect.DeepEqual(got, []uint{0, 1}) {
		t.Errorf("domains got %v, want [0 1]", got)
	}
	if got := topo.CpuInfo().NumNumaDomains; got != 2 {
		t.Errorf("NUMA count got %d, want 2", got)
	}
	root := topologyRegressionRoot(7, 11, 31)
	numa := Object{Type: HWLOC_TYPE_NUMANODE, ID: 4}
	root.Children[0].Children = append(root.Children[0].Children, numa)
	root.Children[0].MemoryChildren = []Object{numa}
	remote, err := RemoteTopology(topologyRegressionJSON(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if got := remote.GetMemoryDomains(); !reflect.DeepEqual(got, []uint{4}) {
		t.Errorf("legacy duplicated NUMA ID got %v, want [4]", got)
	}
	topologyRegressionNUMA(t, remote, 4, []uint{11, 31})
}
