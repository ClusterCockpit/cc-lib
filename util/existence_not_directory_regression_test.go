// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFileExists_NonDirectoryPath(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "file")
	if err := os.WriteFile(file, []byte("owned fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if !CheckFileExists(directory) || !CheckFileExists(file) {
		t.Fatal("existing path lost")
	}
	if CheckFileExists(filepath.Join(directory, "missing")) {
		t.Fatal("missing path reported existing")
	}
	if CheckFileExists(filepath.Join(file, "child")) {
		t.Fatal("child of a regular file reported existing")
	}
}
