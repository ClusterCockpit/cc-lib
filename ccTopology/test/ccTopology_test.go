// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/ccTopology"
)

func TestLocalRemote(t *testing.T) {
	var topo ccTopology.Topology
	topo, err := ccTopology.LocalTopology()
	if err != nil {
		t.Errorf("Failed to init topology: %s", err.Error())
	}
	t.Log("Topology initialized")

	x, err := json.MarshalIndent(topo, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal topology: %s", err.Error())
	}

	newt, err := ccTopology.RemoteTopology(x)
	if err != nil {
		t.Errorf("Failed to unmarshal topology JSON: %s", err.Error())
	}
	y, err := json.MarshalIndent(newt, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal topology: %s", err.Error())
	}
	t.Log(string(y))
}
