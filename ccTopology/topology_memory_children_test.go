// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccTopology

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

func TestTopology_MultipleMemoryChildren(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("native C compiler unavailable")
	}
	source, err := os.ReadFile("ccTopology.go")
	if err != nil {
		t.Fatal(err)
	}
	functions := regexp.MustCompile(`(?s)hwloc_obj_t _hwloc_get_(?:memory|io)_child\([^)]*\) \{.*?\n\}`).FindAll(source, -1)
	if len(functions) != 2 {
		t.Fatal("production C helpers not found")
	}
	fixture := []byte("#include <stdlib.h>\n#include \"hwloc.h\"\n")
	for _, function := range functions {
		fixture = append(fixture, function...)
		fixture = append(fixture, '\n')
	}
	fixture = append(fixture, []byte(`
int main(void) {
 struct hwloc_obj parent={0}, children[3]={{0}};
 parent.memory_arity=2;parent.io_arity=2;
 parent.memory_first_child=&children[0];parent.io_first_child=&children[0];
 children[0].next_sibling=&children[2];
 if (_hwloc_get_memory_child(&parent,0)!=&children[0]) return 1;
 if (_hwloc_get_memory_child(&parent,1)!=&children[2]) return 2;
 if (_hwloc_get_io_child(&parent,1)!=&children[2]) return 3;
 if (_hwloc_get_memory_child(&parent,2)!=NULL) return 4;
 return 0;
}`)...)
	dir := t.TempDir()
	input := filepath.Join(dir, "children.c")
	binary := filepath.Join(dir, "children")
	if err := os.WriteFile(input, fixture, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(cc, "-Ihwloc", "-I.", input, "-o", binary).CombinedOutput(); err != nil {
		t.Fatalf("C fixture compile: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("production child helper returned the wrong sibling: %v\n%s", err, output)
	}
}
