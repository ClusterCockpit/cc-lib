// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"testing"
)

func TestTopologyCpuInfoMissingCores(t *testing.T) {
	for _, root := range []Object{{Type: HWLOC_TYPE_MACHINE}, topologyRegressionRoot(5, 11)} {
		func() {
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("CpuInfo without cores panicked: %v", value)
				}
			}()
			topo, err := RemoteTopology(topologyRegressionJSON(t, root))
			if err != nil {
				t.Fatal(err)
			}
			if info := topo.CpuInfo(); info.NumCores != 0 || info.SMTWidth != 0 {
				t.Errorf("unknown core/SMT values: %+v", info)
			}
		}()
	}
}
