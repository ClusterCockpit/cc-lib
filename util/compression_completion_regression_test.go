// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package util

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCompressionWriteLimitChild(t *testing.T) {
	dir := os.Getenv("R4_GZIP_WRITE_LIMIT_DIR")
	if dir == "" {
		t.Skip("only invoked in isolated child by TestCompressionOutputMustCompleteBeforeDelete")
	}
	data := bytes.Repeat([]byte("ClusterCockpit synthetic archival data\n"), 64)
	in, out := filepath.Join(dir, "input"), filepath.Join(dir, "output.gz")
	if err := os.WriteFile(in, data, 0600); err != nil {
		t.Fatal(err)
	}
	var prepared bytes.Buffer
	w := gzip.NewWriter(&prepared)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	limit := uint64(prepared.Len())
	mode := os.Getenv("R4_GZIP_WRITE_LIMIT_MODE")
	if mode == "flush" {
		limit = 10
	}
	signal.Ignore(syscall.SIGXFSZ)
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: limit, Max: limit}); err != nil {
		t.Fatalf("child-only size limit: %v", err)
	}
	err := CompressFile(in, out)
	_, staterr := os.Stat(in)
	raw, readerr := os.ReadFile(out)
	if readerr != nil {
		t.Fatal(readerr)
	}
	t.Logf("phase=%s limit=%d returned_error=%v original_exists=%v output_bytes=%d", mode, limit, err, staterr == nil, len(raw))
	if err == nil {
		t.Error("unfinished gzip output reported success")
	}
	if staterr != nil {
		t.Error("original removed before output completed")
	}
	if r, e := gzip.NewReader(bytes.NewReader(raw)); e == nil {
		_, e = io.ReadAll(r)
		r.Close()
		if e == nil {
			t.Error("fixture failed to cause incomplete gzip")
		}
	}
}
func TestCompressionOutputMustCompleteBeforeDelete(t *testing.T) {
	for _, mode := range []string{"flush", "footer"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestCompressionWriteLimitChild$", "-test.v")
			cmd.Env = append(os.Environ(), "R4_GZIP_WRITE_LIMIT_DIR="+t.TempDir(), "R4_GZIP_WRITE_LIMIT_MODE="+mode)
			out, err := cmd.CombinedOutput()
			t.Logf("%s", out)
			if err != nil {
				t.Errorf("output-finalization regression: %v", err)
			}
		})
	}
}
func TestCompressionNormalRoundtripAndErrorControls(t *testing.T) {
	dir := t.TempDir()
	in, gz, out := filepath.Join(dir, "input"), filepath.Join(dir, "data.gz"), filepath.Join(dir, "output")
	for _, data := range [][]byte{nil, bytes.Repeat([]byte("normal data"), 128)} {
		if err := os.WriteFile(in, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := CompressFile(in, gz); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(in); !os.IsNotExist(err) {
			t.Fatal("successful compression should remove original")
		}
		if err := UncompressFile(gz, out); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(out)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatal("roundtrip mismatch", err)
		}
		if _, err := os.Stat(gz); !os.IsNotExist(err) {
			t.Fatal("successful decompression should remove input")
		}
	}
	if err := os.WriteFile(in, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompressFile(in, filepath.Join(dir, "absent", "data.gz")); err == nil {
		t.Fatal("missing output parent should error")
	}
	if _, err := os.Stat(in); err != nil {
		t.Fatal("input lost on create error")
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte("incomplete"))
	w.Close()
	if err := os.WriteFile(gz, b.Bytes()[:b.Len()-4], 0600); err != nil {
		t.Fatal(err)
	}
	if err := UncompressFile(gz, out); err == nil {
		t.Fatal("truncated gzip should error")
	}
	if _, err := os.Stat(gz); err != nil {
		t.Fatal("truncated original lost")
	}
}
