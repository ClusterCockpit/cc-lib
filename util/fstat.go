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

// CheckFileExists checks if a file or directory exists at the given path.
// Returns true if the file exists, false otherwise.
func CheckFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

// GetFilesize returns the size of a file in bytes.
// Returns 0 if the file cannot be accessed or does not exist.
func GetFilesize(filePath string) int64 {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		cclog.Errorf("Error on Stat %s: %v", filePath, err)
		return 0
	}
	return fileInfo.Size()
}

// GetFilecount returns the number of entries (files and directories) in a directory.
// Returns 0 if the directory cannot be read.
func GetFilecount(path string) int {
	files, err := os.ReadDir(path)
	if err != nil {
		cclog.Errorf("Error on ReadDir %s: %v", path, err)
		return 0
	}

	return len(files)
}
