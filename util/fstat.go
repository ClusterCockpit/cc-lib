// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package util

import (
	"errors"
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

func CheckFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func GetFilesize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		cclog.Errorf("Error on Stat %s: %v", filePath, err)
		return 0
	}
	return fileInfo.Size()
}

func GetFilecount(path string) int {
	files, err := os.ReadDir(path)
	if err != nil {
		cclog.Errorf("Error on ReadDir %s: %v", path, err)
		return 0
	}

	return len(files)
}
