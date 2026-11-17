// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"reflect"
	"testing"
)

func TestTopologyUnmarshalReplacesState(t *testing.T) {
	var topo topology
	if err := topo.UnmarshalJSON(topologyRegressionJSON(t, topologyRegressionRoot(3, 8, 12))); err != nil {
		t.Fatal(err)
	}
	if err := topo.UnmarshalJSON(topologyRegressionJSON(t, topologyRegressionRoot(9, 40))); err != nil {
		t.Fatal(err)
	}
	if got := topo.GetSockets(); !reflect.DeepEqual(got, []uint{9}) {
		t.Errorf("sockets got %v, want [9]", got)
	}
	if got := topo.GetHwthreads(); !reflect.DeepEqual(got, []uint{40}) {
		t.Errorf("CPUs got %v, want [40]", got)
	}
	if err := topo.UnmarshalJSON([]byte(`{"type":0}`)); err != nil {
		t.Fatal(err)
	}
	if len(topo.GetHwthreads()) != 0 || len(topo.GetSockets()) != 0 {
		t.Error("omitted fields retained old topology")
	}
	before, err := topo.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if err := topo.UnmarshalJSON([]byte(`{"id":17,"children":"invalid"}`)); err == nil {
		t.Fatal("invalid children accepted")
	}
	after, err := topo.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("failed unmarshal changed current topology")
	}
}
