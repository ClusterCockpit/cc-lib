// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"testing"
)

func TestTopologyRemoteNUMAExplicitChildren(t *testing.T) {
	root := Object{Type: HWLOC_TYPE_MACHINE, MemoryChildren: []Object{
		{Type: HWLOC_TYPE_NUMANODE, ID: 4, Children: []Object{{Type: HWLOC_TYPE_PU, ID: 11}}},
		{Type: HWLOC_TYPE_NUMANODE, ID: 19, Children: []Object{{Type: HWLOC_TYPE_PU, ID: 31}}},
	}}
	topo, err := RemoteTopology(topologyRegressionJSON(t, root))
	if err != nil {
		t.Fatal(err)
	}
	topologyRegressionNUMA(t, topo, 4, []uint{11})
	topologyRegressionNUMA(t, topo, 19, []uint{31})
}
