// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package util_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ClusterCockpit/cc-lib/util"
)

func TestCheckFileExists(t *testing.T) {
	tmpdir := t.TempDir()
	if !util.CheckFileExists(tmpdir) {
		t.Fatal("expected true, got false")
	}

	filePath := filepath.Join(tmpdir, "version.txt")

	if err := os.WriteFile(filePath, []byte(fmt.Sprintf("%d", 1)), 0666); err != nil {
		t.Fatal(err)
	}
	if !util.CheckFileExists(filePath) {
		t.Fatal("expected true, got false")
	}

	filePath = filepath.Join(tmpdir, "version-test.txt")
	if util.CheckFileExists(filePath) {
		t.Fatal("expected false, got true")
	}
}

func TestGetFileSize(t *testing.T) {
	tmpdir := t.TempDir()
	filePath := filepath.Join(tmpdir, "data.json")

	if s := util.GetFilesize(filePath); s > 0 {
		t.Fatalf("expected 0, got %d", s)
	}

	if err := os.WriteFile(filePath, []byte(fmt.Sprintf("%d", 1)), 0666); err != nil {
		t.Fatal(err)
	}
	if s := util.GetFilesize(filePath); s == 0 {
		t.Fatal("expected not 0, got 0")
	}
}

func TestGetFileCount(t *testing.T) {
	tmpdir := t.TempDir()

	if c := util.GetFilecount(tmpdir); c != 0 {
		t.Fatalf("expected 0, got %d", c)
	}

	filePath := filepath.Join(tmpdir, "data-1.json")
	if err := os.WriteFile(filePath, []byte(fmt.Sprintf("%d", 1)), 0666); err != nil {
		t.Fatal(err)
	}
	filePath = filepath.Join(tmpdir, "data-2.json")
	if err := os.WriteFile(filePath, []byte(fmt.Sprintf("%d", 1)), 0666); err != nil {
		t.Fatal(err)
	}
	if c := util.GetFilecount(tmpdir); c != 2 {
		t.Fatalf("expected 2, got %d", c)
	}

	if c := util.GetFilecount(filePath); c != 0 {
		t.Fatalf("expected 0, got %d", c)
	}
}

func TestContains(t *testing.T) {
	// Test with integers
	intSlice := []int{1, 2, 3, 4, 5}
	if !util.Contains(intSlice, 3) {
		t.Error("expected Contains to find 3 in slice")
	}
	if util.Contains(intSlice, 10) {
		t.Error("expected Contains to not find 10 in slice")
	}

	// Test with strings
	strSlice := []string{"apple", "banana", "orange"}
	if !util.Contains(strSlice, "banana") {
		t.Error("expected Contains to find 'banana' in slice")
	}
	if util.Contains(strSlice, "grape") {
		t.Error("expected Contains to not find 'grape' in slice")
	}

	// Test with empty slice
	emptySlice := []int{}
	if util.Contains(emptySlice, 1) {
		t.Error("expected Contains to not find anything in empty slice")
	}
}

func TestCompressUncompressFile(t *testing.T) {
	tmpdir := t.TempDir()
	originalFile := filepath.Join(tmpdir, "original.txt")
	compressedFile := filepath.Join(tmpdir, "compressed.gz")
	uncompressedFile := filepath.Join(tmpdir, "uncompressed.txt")

	// Create a test file
	testContent := []byte("This is a test file for compression and decompression.")
	if err := os.WriteFile(originalFile, testContent, 0666); err != nil {
		t.Fatal(err)
	}

	// Test compression
	if err := util.CompressFile(originalFile, compressedFile); err != nil {
		t.Fatalf("CompressFile failed: %v", err)
	}

	// Original file should be removed after compression
	if util.CheckFileExists(originalFile) {
		t.Error("original file should be removed after compression")
	}

	// Compressed file should exist
	if !util.CheckFileExists(compressedFile) {
		t.Error("compressed file should exist")
	}

	// Test decompression
	if err := util.UncompressFile(compressedFile, uncompressedFile); err != nil {
		t.Fatalf("UncompressFile failed: %v", err)
	}

	// Compressed file should be removed after decompression
	if util.CheckFileExists(compressedFile) {
		t.Error("compressed file should be removed after decompression")
	}

	// Verify the content matches
	uncompressedContent, err := os.ReadFile(uncompressedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(uncompressedContent) != string(testContent) {
		t.Errorf("content mismatch: expected %q, got %q", testContent, uncompressedContent)
	}
}

func TestCopyFile(t *testing.T) {
	tmpdir := t.TempDir()
	srcFile := filepath.Join(tmpdir, "source.txt")
	dstFile := filepath.Join(tmpdir, "dest.txt")

	testContent := []byte("Test file content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Test copy
	if err := util.CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination exists
	if !util.CheckFileExists(dstFile) {
		t.Error("destination file should exist after copy")
	}

	// Verify content matches
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(dstContent) != string(testContent) {
		t.Errorf("content mismatch: expected %q, got %q", testContent, dstContent)
	}

	// Verify permissions match
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, _ := os.Stat(dstFile)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions mismatch: expected %v, got %v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	tmpdir := t.TempDir()
	srcDir := filepath.Join(tmpdir, "source")
	dstDir := filepath.Join(tmpdir, "dest")

	// Create source directory structure
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test copy directory
	if err := util.CopyDir(srcDir, dstDir); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Verify files exist
	if !util.CheckFileExists(filepath.Join(dstDir, "file1.txt")) {
		t.Error("file1.txt should exist in destination")
	}
	if !util.CheckFileExists(filepath.Join(dstDir, "subdir", "file2.txt")) {
		t.Error("subdir/file2.txt should exist in destination")
	}

	// Verify content
	content, _ := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if string(content) != "content1" {
		t.Errorf("content mismatch for file1.txt")
	}
	content, _ = os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	if string(content) != "content2" {
		t.Errorf("content mismatch for file2.txt")
	}
}

func TestDiskUsage(t *testing.T) {
	tmpdir := t.TempDir()

	// Empty directory should return 0
	usage := util.DiskUsage(tmpdir)
	if usage != 0.0 {
		t.Errorf("expected 0.0 MB for empty directory, got %f", usage)
	}

	// Create some files
	if err := os.WriteFile(filepath.Join(tmpdir, "file1.txt"), make([]byte, 1000000), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpdir, "file2.txt"), make([]byte, 500000), 0644); err != nil {
		t.Fatal(err)
	}

	// Should return approximately 1.5 MB
	usage = util.DiskUsage(tmpdir)
	if usage < 1.4 || usage > 1.6 {
		t.Errorf("expected ~1.5 MB, got %f", usage)
	}
}

func TestFloat(t *testing.T) {
	// Test IsNaN
	f := util.NaN
	if !f.IsNaN() {
		t.Error("expected NaN.IsNaN() to return true")
	}

	f = util.Float(3.14)
	if f.IsNaN() {
		t.Error("expected Float(3.14).IsNaN() to return false")
	}

	// Test Double
	if f.Double() != 3.14 {
		t.Errorf("expected Double() to return 3.14, got %f", f.Double())
	}

	// Test MarshalJSON for NaN
	nanJSON, err := util.NaN.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(nanJSON) != "null" {
		t.Errorf("expected 'null' for NaN, got %s", nanJSON)
	}

	// Test MarshalJSON for normal value
	normalJSON, err := util.Float(3.142).MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(normalJSON) != "3.142" {
		t.Errorf("expected '3.142', got %s", normalJSON)
	}

	// Test UnmarshalJSON for null
	var f2 util.Float
	if err := f2.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !f2.IsNaN() {
		t.Error("expected unmarshaled null to be NaN")
	}

	// Test UnmarshalJSON for number
	if err := f2.UnmarshalJSON([]byte("5.678")); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if f2.Double() != 5.678 {
		t.Errorf("expected 5.678, got %f", f2.Double())
	}

	// Test ConvertToFloat
	converted := util.ConvertToFloat(-1.0)
	if !converted.IsNaN() {
		t.Error("expected ConvertToFloat(-1.0) to return NaN")
	}
	converted = util.ConvertToFloat(10.5)
	if converted.Double() != 10.5 {
		t.Errorf("expected ConvertToFloat(10.5) to return 10.5, got %f", converted.Double())
	}

	// Test FloatArray MarshalJSON
	arr := util.FloatArray{util.Float(1.0), util.NaN, util.Float(3.0)}
	arrJSON, err := arr.MarshalJSON()
	if err != nil {
		t.Fatalf("FloatArray MarshalJSON failed: %v", err)
	}
	if string(arrJSON) != "[1.000,null,3.000]" {
		t.Errorf("expected '[1.000,null,3.000]', got %s", arrJSON)
	}
}

func TestSelectorElement(t *testing.T) {
	// Test UnmarshalJSON for string
	var se util.SelectorElement
	if err := json.Unmarshal([]byte(`"test"`), &se); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if se.String != "test" {
		t.Errorf("expected String='test', got %q", se.String)
	}

	// Test UnmarshalJSON for wildcard
	if err := json.Unmarshal([]byte(`"*"`), &se); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if !se.Any || se.String != "" {
		t.Error("expected Any=true and String='' for wildcard")
	}

	// Test UnmarshalJSON for array
	if err := json.Unmarshal([]byte(`["a","b","c"]`), &se); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if len(se.Group) != 3 || se.Group[0] != "a" || se.Group[1] != "b" || se.Group[2] != "c" {
		t.Errorf("expected Group=['a','b','c'], got %v", se.Group)
	}

	// Test MarshalJSON for Any
	se = util.SelectorElement{Any: true}
	data, err := json.Marshal(&se)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(data) != `"*"` {
		t.Errorf("expected '\"*\"', got %s", data)
	}

	// Test MarshalJSON for String
	se = util.SelectorElement{String: "test"}
	data, err = json.Marshal(&se)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(data) != `"test"` {
		t.Errorf("expected '\"test\"', got %s", data)
	}

	// Test MarshalJSON for Group
	se = util.SelectorElement{Group: []string{"x", "y"}}
	data, err = json.Marshal(&se)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if string(data) != `["x","y"]` {
		t.Errorf("expected '[\"x\",\"y\"]', got %s", data)
	}
}

func TestMin(t *testing.T) {
	if util.Min(5, 3) != 3 {
		t.Error("Min(5, 3) should return 3")
	}
	if util.Min(2, 8) != 2 {
		t.Error("Min(2, 8) should return 2")
	}
	if util.Min("apple", "banana") != "apple" {
		t.Error("Min(apple, banana) should return apple")
	}
}

func TestMax(t *testing.T) {
	if util.Max(5, 3) != 5 {
		t.Error("Max(5, 3) should return 5")
	}
	if util.Max(2, 8) != 8 {
		t.Error("Max(2, 8) should return 8")
	}
	if util.Max("apple", "banana") != "banana" {
		t.Error("Max(apple, banana) should return banana")
	}
}

func TestMean(t *testing.T) {
	// Test with normal values
	mean, err := util.Mean([]float64{1.0, 2.0, 3.0, 4.0, 5.0})
	if err != nil {
		t.Fatalf("Mean failed: %v", err)
	}
	if mean != 3.0 {
		t.Errorf("expected mean=3.0, got %f", mean)
	}

	// Test with single value
	mean, err = util.Mean([]float64{7.5})
	if err != nil {
		t.Fatalf("Mean failed: %v", err)
	}
	if mean != 7.5 {
		t.Errorf("expected mean=7.5, got %f", mean)
	}

	// Test with empty slice
	mean, err = util.Mean([]float64{})
	if err == nil {
		t.Error("expected error for empty slice")
	}
	if !math.IsNaN(mean) {
		t.Error("expected NaN for empty slice")
	}
}

func TestMedian(t *testing.T) {
	// Test with odd number of elements
	median, err := util.Median([]float64{1.0, 3.0, 2.0, 5.0, 4.0})
	if err != nil {
		t.Fatalf("Median failed: %v", err)
	}
	if median != 3.0 {
		t.Errorf("expected median=3.0, got %f", median)
	}

	// Test with even number of elements
	median, err = util.Median([]float64{1.0, 2.0, 3.0, 4.0})
	if err != nil {
		t.Fatalf("Median failed: %v", err)
	}
	if median != 2.5 {
		t.Errorf("expected median=2.5, got %f", median)
	}

	// Test with single value
	median, err = util.Median([]float64{7.5})
	if err != nil {
		t.Fatalf("Median failed: %v", err)
	}
	if median != 7.5 {
		t.Errorf("expected median=7.5, got %f", median)
	}

	// Test with empty slice
	median, err = util.Median([]float64{})
	if err == nil {
		t.Error("expected error for empty slice")
	}
	if !math.IsNaN(median) {
		t.Error("expected NaN for empty slice")
	}
}
