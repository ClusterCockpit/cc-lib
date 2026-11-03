// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"testing"
)

func TestTopologyRemoteNUMASparseGroups(t *testing.T) {
	root := Object{Type: HWLOC_TYPE_MACHINE, Children: []Object{{Type: HWLOC_TYPE_PACKAGE, ID: 5, Children: []Object{
		{Type: HWLOC_TYPE_GROUP, Children: []Object{{Type: HWLOC_TYPE_PU, ID: 11}, {Type: HWLOC_TYPE_PU, ID: 31}}, MemoryChildren: []Object{{Type: HWLOC_TYPE_NUMANODE, ID: 4}}},
		{Type: HWLOC_TYPE_GROUP, Children: []Object{{Type: HWLOC_TYPE_PU, ID: 63}}, MemoryChildren: []Object{{Type: HWLOC_TYPE_NUMANODE, ID: 19}}},
	}}}}
	topo, err := RemoteTopology(topologyRegressionJSON(t, root))
	if err != nil {
		t.Fatal(err)
	}
	topologyRegressionNUMA(t, topo, 4, []uint{11, 31})
	topologyRegressionNUMA(t, topo, 19, []uint{63})
}
