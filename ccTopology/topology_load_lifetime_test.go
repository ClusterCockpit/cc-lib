// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTopology_LoadFailureReleasesContext(t *testing.T) {
	if os.Getenv("CC_TOPOLOGY_LIFETIME_CHILD") != "1" {
		if runtime.GOOS != "linux" {
			t.Skip("native lifetime interposition requires Linux")
		}
		cc, err := exec.LookPath("cc")
		if err != nil {
			t.Skip("native C compiler unavailable")
		}
		dir := t.TempDir()
		probe := filepath.Join(dir, "probe.so")
		trace := filepath.Join(dir, "trace.json")
		build := exec.Command(cc, "-shared", "-fPIC", "-Ihwloc", "-I.", "testdata/hwloc_lifetime_probe.c", "-ldl", "-o", probe)
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("compile native probe: %v\n%s", err, output)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestTopology_LoadFailureReleasesContext$", "-test.count=1")
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "LD_PRELOAD=") && !strings.HasPrefix(entry, "CC_TOPOLOGY_LIFETIME_") {
				child.Env = append(child.Env, entry)
			}
		}
		child.Env = append(child.Env, "LD_PRELOAD="+probe, "CC_TOPOLOGY_LIFETIME_TRACE="+trace, "CC_TOPOLOGY_LIFETIME_CHILD=1")
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("native lifetime regression: %v\n%s", err, output)
		}
		return
	}
	t.Setenv("HWLOC_SYNTHETIC", "")
	xml := filepath.Join(t.TempDir(), "truncated.xml")
	if err := os.WriteFile(xml, []byte(`<topology version="2.0">`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HWLOC_XMLFILE", xml)
	_ccTopology_local_topo = nil
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := LocalTopology(); err == nil {
			t.Fatal("truncated XML unexpectedly loaded")
		}
		data, err := os.ReadFile(os.Getenv("CC_TOPOLOGY_LIFETIME_TRACE"))
		if err != nil {
			t.Fatal(err)
		}
		var counts struct{ Initialized, Failed, Loaded, Destroyed int }
		if err := json.Unmarshal(data, &counts); err != nil {
			t.Fatal(err)
		}
		if counts.Initialized != attempt || counts.Failed != attempt || counts.Destroyed != attempt || counts.Loaded != 0 {
			t.Errorf("failed attempt %d leaked native context: %+v", attempt, counts)
		}
	}
	t.Setenv("HWLOC_XMLFILE", "")
	t.Setenv("HWLOC_SYNTHETIC", "package:1 numa:1 core:1 pu:2")
	topo, err := LocalTopology()
	if err != nil {
		t.Fatal(err)
	}
	if cpus := topo.GetHwthreadsOfMemoryDomain(0); len(cpus) != 2 {
		t.Fatalf("successful context is not live: %v", cpus)
	}
}
