// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package util

import (
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

func DiskUsage(dirpath string) float64 {
	var size int64

	dir, err := os.Open(dirpath)
	if err != nil {
		cclog.Errorf("DiskUsage() error: %v", err)
		return 0
	}
	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		cclog.Errorf("DiskUsage() error: %v", err)
		return 0
	}

	for _, file := range files {
		size += file.Size()
	}

	return float64(size) * 1e-6
}
