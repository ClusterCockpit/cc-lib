package main

import (
	"encoding/json"

	"testing"

	"github.com/ClusterCockpit/cc-lib/ccTopology"
)

func TestLocalRemote(t *testing.T) {
	var topo ccTopology.Topology
	topo, err := ccTopology.LocalTopology()
	if err != nil {
		t.Errorf("Failed to init topology: %v", err.Error())
	}
	t.Log("Topology initialized")

	x, err := json.MarshalIndent(topo, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal topology: %v", err.Error())
	}

	newt, err := ccTopology.RemoteTopology(x)
	if err != nil {
		t.Errorf("Failed to unmarshal topology JSON: %v", err.Error())
	}
	y, err := json.MarshalIndent(newt, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal topology: %v", err.Error())
	}
	t.Log(string(y))
}
