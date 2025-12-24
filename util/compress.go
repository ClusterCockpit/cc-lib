// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"compress/gzip"
	"io"
	"os"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// CompressFile compresses a file using gzip compression.
// It reads the input file (fileIn), creates a gzip-compressed output file (fileOut),
// and removes the original file upon successful compression.
// Returns an error if any operation fails.
func CompressFile(fileIn string, fileOut string) error {
	originalFile, err := os.Open(fileIn)
	if err != nil {
		cclog.Errorf("CompressFile() error: %v", err)
		return err
	}
	defer originalFile.Close()

	gzippedFile, err := os.Create(fileOut)
	if err != nil {
		cclog.Errorf("CompressFile() error: %v", err)
		return err
	}
	defer gzippedFile.Close()

	gzipWriter := gzip.NewWriter(gzippedFile)
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, originalFile)
	if err != nil {
		cclog.Errorf("CompressFile() error: %v", err)
		return err
	}
	gzipWriter.Flush()
	if err := os.Remove(fileIn); err != nil {
		cclog.Errorf("CompressFile() error: %v", err)
		return err
	}

	return nil
}

// UncompressFile decompresses a gzip-compressed file.
// It reads the gzip-compressed input file (fileIn), creates an uncompressed output file (fileOut),
// and removes the compressed file upon successful decompression.
// Returns an error if any operation fails.
func UncompressFile(fileIn string, fileOut string) error {
	gzippedFile, err := os.Open(fileIn)
	if err != nil {
		cclog.Errorf("UncompressFile() error: %v", err)
		return err
	}
	defer gzippedFile.Close()

	gzipReader, err := gzip.NewReader(gzippedFile)
	if err != nil {
		cclog.Errorf("UncompressFile() error creating gzip reader: %v", err)
		return err
	}
	defer gzipReader.Close()

	uncompressedFile, err := os.Create(fileOut)
	if err != nil {
		cclog.Errorf("UncompressFile() error: %v", err)
		return err
	}
	defer uncompressedFile.Close()

	_, err = io.Copy(uncompressedFile, gzipReader)
	if err != nil {
		cclog.Errorf("UncompressFile() error: %v", err)
		return err
	}
	if err := os.Remove(fileIn); err != nil {
		cclog.Errorf("UncompressFile() error: %v", err)
		return err
	}

	return nil
}
