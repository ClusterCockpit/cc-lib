// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

func topologyRegressionJSON(t *testing.T, root Object) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func topologyRegressionRoot(socket uint, cpus ...uint) Object {
	pkg := Object{Type: HWLOC_TYPE_PACKAGE, ID: socket}
	for _, id := range cpus {
		pkg.Children = append(pkg.Children, Object{Type: HWLOC_TYPE_PU, ID: id, IDString: strconv.FormatUint(uint64(id), 10)})
	}
	return Object{Type: HWLOC_TYPE_MACHINE, Children: []Object{pkg}}
}

func topologyRegressionLocal(t *testing.T, description string) Topology {
	t.Helper()
	t.Setenv("HWLOC_SYNTHETIC", description)
	_ccTopology_local_topo = nil
	topo, err := LocalTopology()
	if err != nil {
		t.Fatal(err)
	}
	return topo
}

func topologyRegressionNUMA(t *testing.T, topo Topology, domain uint, want []uint) {
	t.Helper()
	defer func() {
		if value := recover(); value != nil {
			t.Errorf("NUMA getter panicked: %v", value)
		}
	}()
	if got := topo.GetHwthreadsOfMemoryDomain(domain); !reflect.DeepEqual(got, want) {
		t.Errorf("domain %d CPUs got %v, want %v", domain, got, want)
	}
	strings := make([]string, 0, len(want))
	for _, id := range want {
		strings = append(strings, strconv.FormatUint(uint64(id), 10))
	}
	if got := topo.GetHwthreadStringsOfMemoryDomain(domain); !reflect.DeepEqual(got, strings) {
		t.Errorf("domain %d CPU strings got %v, want %v", domain, got, strings)
	}
}
