// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"testing"
)

func TestTopologyRemoteNUMAAmbiguous(t *testing.T) {
	root := topologyRegressionRoot(5, 11, 31)
	root.Children[0].MemoryChildren = []Object{{Type: HWLOC_TYPE_NUMANODE, ID: 4}, {Type: HWLOC_TYPE_NUMANODE, ID: 19}}
	topo, err := RemoteTopology(topologyRegressionJSON(t, root))
	if err != nil {
		t.Fatal(err)
	}
	// JSON without cpusets cannot distinguish two domains at the same CPU ancestor.
	// An empty result represents unavailable membership instead of invented CPUs.
	topologyRegressionNUMA(t, topo, 4, []uint{})
	topologyRegressionNUMA(t, topo, 19, []uint{})
}
