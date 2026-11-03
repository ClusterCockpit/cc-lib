// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"reflect"
	"slices"
	"testing"
)

func TestTopologyCoverage_DuplicateSelections(t *testing.T) {
	topo := Topology{Node: []int{10, 20, 30, 40}, Socket: [][]int{{10, 20, 30, 40}}, Core: [][]int{{10, 20}, {30, 40}}, MemoryDomain: [][]int{{10, 20}, {30, 40}}}
	topo.InitTopologyMaps()
	for _, tc := range []struct {
		name  string
		get   func([]int) ([]int, bool)
		input []int
		want  bool
	}{
		{"socket missing members", topo.GetSocketsFromHWThreads, []int{10, 10, 10, 10}, false},
		{"core missing member", topo.GetCoresFromHWThreads, []int{10, 10}, false},
		{"NUMA missing member", topo.GetMemoryDomainsFromHWThreads, []int{10, 10}, false},
		{"socket missing core", topo.GetSocketsFromCores, []int{0, 0}, false},
		{"complete socket repeated", topo.GetSocketsFromHWThreads, []int{10, 20, 30, 40, 10}, true},
		{"complete core repeated", topo.GetCoresFromHWThreads, []int{10, 20, 10}, true},
		{"complete NUMA repeated", topo.GetMemoryDomainsFromHWThreads, []int{10, 20, 10}, true},
		{"complete core set repeated", topo.GetSocketsFromCores, []int{0, 1, 0}, true},
	} {
		inputBefore := append([]int(nil), tc.input...)
		got, exclusive := tc.get(tc.input)
		if !reflect.DeepEqual(got, []int{0}) || exclusive != tc.want {
			t.Errorf("%s input=%v groups=%v exclusive=%v want=%v", tc.name, tc.input, got, exclusive, tc.want)
		}
		if !reflect.DeepEqual(tc.input, inputBefore) {
			t.Errorf("input changed: %v -> %v", inputBefore, tc.input)
		}
	}
}

func TestTopologyCoverage_Compatibility(t *testing.T) {
	{
		topo := Topology{Node: []int{10, 20, 30, 40}}
		topo.Socket = make([][]int, 8)
		topo.Socket[2], topo.Socket[7] = []int{10, 20}, []int{30, 40}
		topo.Core = make([][]int, 12)
		topo.Core[5], topo.Core[11] = []int{10, 20}, []int{30, 40}
		topo.MemoryDomain = make([][]int, 10)
		topo.MemoryDomain[3], topo.MemoryDomain[9] = []int{10, 20}, []int{30, 40}
		topo.InitTopologyMaps()
		for _, tc := range []struct {
			name        string
			get         func([]int) ([]int, bool)
			input, want []int
			exclusive   bool
		}{
			{"socket partial", topo.GetSocketsFromHWThreads, []int{10}, []int{2}, false},
			{"socket complete", topo.GetSocketsFromHWThreads, []int{10, 20}, []int{2}, true},
			{"all sockets", topo.GetSocketsFromHWThreads, []int{10, 20, 30, 40}, []int{2, 7}, true},
			{"core partial", topo.GetCoresFromHWThreads, []int{30}, []int{11}, false},
			{"core complete", topo.GetCoresFromHWThreads, []int{30, 40}, []int{11}, true},
			{"socket from physical core", topo.GetSocketsFromCores, []int{11}, []int{7}, true},
			{"NUMA partial", topo.GetMemoryDomainsFromHWThreads, []int{10}, []int{3}, false},
			{"NUMA complete", topo.GetMemoryDomainsFromHWThreads, []int{30, 40}, []int{9}, true},
		} {
			got, exclusive := tc.get(tc.input)
			slices.Sort(got)
			if !reflect.DeepEqual(got, tc.want) || exclusive != tc.exclusive {
				t.Errorf("%s: got=%v exclusive=%v want=%v/%v", tc.name, got, exclusive, tc.want, tc.exclusive)
			}
		}
		groups, err := topo.GetMemoryDomainsBySocket([]int{9, 3})
		if err != nil || !reflect.DeepEqual(groups, map[int][]int{2: {3}, 7: {9}}) {
			t.Fatalf("groups=%v err=%v", groups, err)
		}

	}
	{
		topo := Topology{Socket: [][]int{{10, 20}}, Core: [][]int{{10}, {20}}, MemoryDomain: [][]int{{10, 20}, {}, {99}}}
		before, exclusive := topo.GetSocketsFromHWThreads([]int{10})
		if len(before) != 0 || !exclusive {
			t.Fatal("uninitialized lookup convention changed")
		}
		topo.InitTopologyMaps()
		for _, lookup := range []func([]int) ([]int, bool){topo.GetSocketsFromHWThreads, topo.GetSocketsFromCores, topo.GetCoresFromHWThreads, topo.GetMemoryDomainsFromHWThreads} {
			for _, input := range [][]int{nil, {}, {999}, {999, 999}} {
				got, exclusive := lookup(input)
				if len(got) != 0 || !exclusive {
					t.Fatalf("empty/unknown selection=%v got=%v exclusive=%v", input, got, exclusive)
				}
			}
		}
		for _, input := range [][]int{{-1}, {1}, {2}, {3}} {
			if _, err := topo.GetMemoryDomainsBySocket(input); err == nil {
				t.Errorf("invalid/empty/unmapped NUMA=%v accepted", input)
			}
		}
		groups, err := topo.GetMemoryDomainsBySocket(nil)
		if err != nil || len(groups) != 0 {
			t.Fatalf("empty grouping=%v err=%v", groups, err)
		}
		groups, err = topo.GetMemoryDomainsBySocket([]int{0, 0})
		if err != nil || !reflect.DeepEqual(groups, map[int][]int{0: {0, 0}}) {
			t.Fatalf("grouping multiset convention=%v err=%v", groups, err)
		}
		// Init is currently a one-time cache fill, not a mutation invalidation API.
		topo.Socket[0] = []int{30}
		topo.InitTopologyMaps()
		got, _ := topo.GetSocketsFromHWThreads([]int{10})
		if !reflect.DeepEqual(got, []int{0}) {
			t.Fatalf("one-time cache convention changed: %v", got)
		}
		var empty Topology
		empty.InitTopologyMaps()
		if got, exclusive := empty.GetCoresFromHWThreads(nil); len(got) != 0 || !exclusive {
			t.Fatal("zero topology convention")
		}

	}
	{
		topo := Topology{Accelerators: []*Accelerator{{ID: "0"}, {ID: "4"}, {ID: "GPU-0"}}}
		if id, err := topo.GetAcceleratorID(1); err != nil || id != "4" {
			t.Fatalf("index versus ID: %q %v", id, err)
		}
		for _, index := range []int{-1, 3} {
			if _, err := topo.GetAcceleratorID(index); err == nil {
				t.Errorf("index %d accepted", index)
			}
		}
		if !reflect.DeepEqual(topo.GetAcceleratorIDs(), []string{"0", "4", "GPU-0"}) {
			t.Fatal("string IDs changed")
		}
		if _, err := topo.GetAcceleratorIDsAsInt(); err == nil {
			t.Fatal("nonnumeric ID accepted")
		}
		topo.Accelerators = topo.Accelerators[:2]
		if ids, err := topo.GetAcceleratorIDsAsInt(); err != nil || !reflect.DeepEqual(ids, []int{0, 4}) {
			t.Fatalf("numeric IDs=%v err=%v", ids, err)
		}
		var empty Topology
		if len(empty.GetAcceleratorIDs()) != 0 {
			t.Fatal("empty IDs")
		}
		if ids, err := empty.GetAcceleratorIDsAsInt(); err != nil || len(ids) != 0 {
			t.Fatalf("empty numeric IDs=%v err=%v", ids, err)
		}

	}
}
