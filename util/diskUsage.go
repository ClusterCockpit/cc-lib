// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// DiskUsage calculates the total disk usage of a directory in megabytes (MB).
// It sums up the sizes of all files in the specified directory (non-recursive)
// and returns the result in MB (multiplied by 1e-6).
// Returns 0 if the directory cannot be opened or read.
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
