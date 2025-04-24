// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package cclogger

import (
	"testing"
)

func TestInit(t *testing.T) {
	Init("info", false)

	if lvl := Loglevel(); lvl != "info" {
		t.Errorf("Wrong loglevel got: %s \nwant: info", lvl)
	}
}

func TestOutfile(t *testing.T) {
	dir := t.TempDir()
	Init("info", false)

	SetOutputFile("error", dir+"output.log")
	Info("It worked 1")
	Info("It worked 2")
	Error("It worked 3")
}
