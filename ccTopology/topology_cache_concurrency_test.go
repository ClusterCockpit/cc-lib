// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"sync"
	"testing"
)

func TestTopologyLocalCacheConcurrent(t *testing.T) {
	t.Setenv("HWLOC_SYNTHETIC", "package:2 numa:1 core:2 pu:2")
	_ccTopology_local_topo = nil
	const count = 8
	start := make(chan struct{})
	results := make([]Topology, count)
	var done sync.WaitGroup
	done.Add(count)
	for i := range count {
		go func() {
			defer done.Done()
			<-start
			topo, err := LocalTopology()
			if err != nil {
				t.Errorf("LocalTopology: %v", err)
				return
			}
			results[i] = topo
			if len(topo.GetHwthreads()) != 8 {
				t.Error("unexpected synthetic CPU count")
			}
		}()
	}
	close(start)
	done.Wait()
	for i, topo := range results {
		if topo != results[0] {
			t.Errorf("caller %d received a different cached topology", i)
		}
	}
}
