// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"testing"
)

func TestTopology_SelectsRequestedMemoryDomain(t *testing.T) {
	topo := topologyRegressionLocal(t, "package:2 numa:1 core:2 pu:2")
	if got := topo.GetHwthreadsOfMemoryDomain(999); len(got) != 0 {
		t.Fatalf("unknown domain returned %v", got)
	}
	a, b := topo.GetHwthreadsOfMemoryDomain(0), topo.GetHwthreadsOfMemoryDomain(1)
	setA, setB := map[uint]bool{}, map[uint]bool{}
	for _, v := range a {
		setA[v] = true
	}
	for _, v := range b {
		setB[v] = true
	}
	if len(setA) != 4 || len(setB) != 4 {
		t.Fatalf("domain lengths %v / %v", a, b)
	}
	seen := map[uint]bool{}
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		if seen[v] {
			t.Fatalf("domains overlap at %d", v)
		}
	}
}
