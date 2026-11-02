// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util_test

import (
	"bytes"
	"compress/gzip"
	"github.com/ClusterCockpit/cc-lib/v2/util"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTransformsRejectSameSourceInode(t *testing.T) {
	for _, transform := range []struct {
		name       string
		apply      func(string, string) error
		compressed bool
	}{
		{"copy", util.CopyFile, false}, {"compress", util.CompressFile, false}, {"uncompress", util.UncompressFile, true},
	} {
		for _, alias := range []string{"same path", "hardlink", "symlink"} {
			t.Run(transform.name+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				source := filepath.Join(dir, "source")
				original := []byte("audit file content")
				if transform.compressed {
					var buf bytes.Buffer
					writer := gzip.NewWriter(&buf)
					if _, err := writer.Write(original); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					original = buf.Bytes()
				}
				if err := os.WriteFile(source, original, 0600); err != nil {
					t.Fatal(err)
				}
				destination := source
				if alias != "same path" {
					destination = filepath.Join(dir, "alias")
					var err error
					if alias == "hardlink" {
						err = os.Link(source, destination)
					} else {
						err = os.Symlink(source, destination)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := transform.apply(source, destination); err == nil {
					t.Error("same inode was accepted")
				}
				got, err := os.ReadFile(source)
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("source was lost or modified: %q, %v", got, err)
				}
			})
		}
	}
}
