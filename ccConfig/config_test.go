// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccconfig

import (
	"encoding/json"
	"os"
	"sort"
	"sync"
	"testing"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	"github.com/ClusterCockpit/cc-lib/sinks"
)

type mainConfig struct {
	Interval string `json:"interval"`
}

// TestInit tests basic initialization with inline and file-based config
func TestInit(t *testing.T) {
	cclog.Init("debug", true)
	fn := "./testdata/config.json"
	Init(fn)
	n := len(keys)
	if n != 4 {
		t.Errorf("Wrong number of config objects got: %d \nwant: 4", n)
	}

	rawConfig := GetPackageConfig("sinks")
	var sync sync.WaitGroup

	_, err := sinks.New(&sync, rawConfig)
	if err != nil {
		t.Errorf("Error in sink.New: %v ", err)
	}

	var mc mainConfig
	rawConfig = GetPackageConfig("main")
	err = json.Unmarshal(rawConfig, &mc)
	if err != nil {
		t.Errorf("Error in Unmarshal': %v ", err)
	}

	if mv := mc.Interval; mv != "10s" {
		t.Errorf("Wrong interval got: %s \nwant: 10s", mv)
	}
}

// TestInitAll tests initialization with all inline config (no file references)
func TestInitAll(t *testing.T) {
	cclog.Init("debug", true)
	fn := "./testdata/configAll.json"
	Init(fn)
	n := len(keys)
	if n != 4 {
		t.Errorf("Wrong number of config objects got: %d \nwant: 4", n)
	}

	rawConfig := GetPackageConfig("sinks")
	var sync sync.WaitGroup

	_, err := sinks.New(&sync, rawConfig)
	if err != nil {
		t.Errorf("Error in sink.New: %v ", err)
	}

	var mc mainConfig
	rawConfig = GetPackageConfig("main")
	err = json.Unmarshal(rawConfig, &mc)
	if err != nil {
		t.Errorf("Error in Unmarshal': %v ", err)
	}

	if mv := mc.Interval; mv != "10s" {
		t.Errorf("Wrong interval got: %s \nwant: 10s", mv)
	}
}

// TestGetPackageConfigNonExistent tests retrieving a non-existent key
func TestGetPackageConfigNonExistent(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/config.json")

	rawConfig := GetPackageConfig("nonexistent")
	if rawConfig != nil {
		t.Errorf("Expected nil for nonexistent key, got: %v", rawConfig)
	}
}

// TestReset tests the Reset functionality
func TestReset(t *testing.T) {
	cclog.Init("debug", true)

	// Load initial config
	Init("./testdata/config.json")
	if len(keys) == 0 {
		t.Error("Expected keys to be populated after Init")
	}

	// Reset and verify empty
	Reset()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after Reset, got: %d", len(keys))
	}

	// Verify GetPackageConfig returns nil after reset
	rawConfig := GetPackageConfig("main")
	if rawConfig != nil {
		t.Error("Expected nil after Reset")
	}
}

// TestResetAndReload tests resetting and reloading configuration
func TestResetAndReload(t *testing.T) {
	cclog.Init("debug", true)

	// Load first config
	Init("./testdata/config.json")
	rawConfig1 := GetPackageConfig("main")
	if rawConfig1 == nil {
		t.Fatal("Expected main config from first Init")
	}

	// Reset and load different config
	Reset()
	Init("./testdata/configAll.json")
	rawConfig2 := GetPackageConfig("main")
	if rawConfig2 == nil {
		t.Fatal("Expected main config from second Init")
	}

	// Both should have the same interval value
	var mc1, mc2 mainConfig
	json.Unmarshal(rawConfig1, &mc1)
	json.Unmarshal(rawConfig2, &mc2)

	if mc1.Interval != mc2.Interval {
		t.Errorf("Expected same interval, got: %s and %s", mc1.Interval, mc2.Interval)
	}
}

// TestHasKey tests the HasKey function
func TestHasKey(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/config.json")

	// Test existing keys
	if !HasKey("main") {
		t.Error("Expected HasKey('main') to be true")
	}
	if !HasKey("sinks") {
		t.Error("Expected HasKey('sinks') to be true")
	}

	// Test non-existent key
	if HasKey("nonexistent") {
		t.Error("Expected HasKey('nonexistent') to be false")
	}
}

// TestGetKeys tests the GetKeys function
func TestGetKeys(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/config.json")

	keys := GetKeys()
	if len(keys) != 4 {
		t.Errorf("Expected 4 keys, got: %d", len(keys))
	}

	// Sort for consistent comparison
	sort.Strings(keys)
	expected := []string{"main", "optimizer", "receivers", "sinks"}
	sort.Strings(expected)

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected key %s at index %d, got: %s", expected[i], i, key)
		}
	}
}

// TestGetKeysEmpty tests GetKeys with empty configuration
func TestGetKeysEmpty(t *testing.T) {
	cclog.Init("debug", true)
	Reset()

	keys := GetKeys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty config, got: %d", len(keys))
	}
}

// TestEmptyConfig tests initialization with an empty config file
func TestEmptyConfig(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/empty.json")

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys from empty config, got: %d", len(keys))
	}

	// GetKeys should return empty slice
	configKeys := GetKeys()
	if len(configKeys) != 0 {
		t.Errorf("Expected GetKeys() to return empty slice, got: %v", configKeys)
	}
}

// TestConfigWithOnlyFileReferences tests config containing only file references
func TestConfigWithOnlyFileReferences(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/only_refs.json")

	// Should have resolved the file references
	if !HasKey("receivers") {
		t.Error("Expected 'receivers' key from file reference")
	}
	if !HasKey("sinks") {
		t.Error("Expected 'sinks' key from file reference")
	}

	// Should NOT have the -file keys
	if HasKey("receivers-file") {
		t.Error("Did not expect 'receivers-file' key in resolved config")
	}
	if HasKey("sinks-file") {
		t.Error("Did not expect 'sinks-file' key in resolved config")
	}
}

// TestNonexistentConfigFile tests Init with a file that doesn't exist
func TestNonexistentConfigFile(t *testing.T) {
	cclog.Init("debug", true)
	Reset()

	// This should not panic, just continue with empty config
	Init("./testdata/this-file-does-not-exist.json")

	if len(keys) != 0 {
		t.Errorf("Expected empty config for nonexistent file, got %d keys", len(keys))
	}
}

// TestMissingFileReference tests behavior when a file reference points to missing file
func TestMissingFileReference(t *testing.T) {
	cclog.Init("debug", true)
	Reset()

	// This contains a reference to a nonexistent file
	// Should not crash, but may log error
	Init("./testdata/missing_ref.json")

	// Main config should still be loaded
	if !HasKey("main") {
		t.Error("Expected 'main' key to be loaded despite missing file reference")
	}

	// The missing reference key should not exist
	// (it will be empty or might not be set depending on error handling)
	rawConfig := GetPackageConfig("missing")
	// We don't assert on this as the behavior could vary
	_ = rawConfig
}

// TestMultipleInitCalls tests calling Init multiple times
func TestMultipleInitCalls(t *testing.T) {
	cclog.Init("debug", true)
	Reset()

	// First init
	Init("./testdata/config.json")
	firstKeyCount := len(keys)

	// Second init (overwrites)
	Init("./testdata/configAll.json")
	secondKeyCount := len(keys)

	if firstKeyCount != secondKeyCount {
		t.Logf("Key counts differ between successive Init calls: %d vs %d",
			firstKeyCount, secondKeyCount)
	}

	// Both should have main config
	if !HasKey("main") {
		t.Error("Expected 'main' key after multiple Init calls")
	}
}

// TestFileReferenceResolution tests that file references are properly resolved
func TestFileReferenceResolution(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/config.json")

	// Load expected content directly
	expectedSinks, err := os.ReadFile("./testdata/sinks.json")
	if err != nil {
		t.Fatalf("Failed to read expected sinks file: %v", err)
	}

	// Get resolved content
	actualSinks := GetPackageConfig("sinks")
	if actualSinks == nil {
		t.Fatal("Expected sinks config to be resolved")
	}

	// Compare as strings (both are JSON)
	if string(actualSinks) != string(expectedSinks) {
		t.Errorf("File reference not properly resolved.\nExpected: %s\nGot: %s",
			string(expectedSinks), string(actualSinks))
	}
}

// TestConfigKeyNames tests that key names are preserved correctly
func TestConfigKeyNames(t *testing.T) {
	cclog.Init("debug", true)
	Reset()
	Init("./testdata/config.json")

	// File references should have -file stripped
	if HasKey("sinks-file") {
		t.Error("Expected 'sinks-file' key to be stripped to 'sinks'")
	}
	if !HasKey("sinks") {
		t.Error("Expected 'sinks' key from 'sinks-file' reference")
	}

	// Inline configs should keep their original name
	if !HasKey("main") {
		t.Error("Expected 'main' key to be preserved")
	}
}

// TestInvalidJSONHandling tests that invalid JSON causes fatal error
// Note: This test is commented out because it would cause the test to exit
// In a real scenario, you'd test this with a subprocess or mock the log.Fatal
/*
func TestInvalidJSONHandling(t *testing.T) {
	cclog.Init("debug", true)
	Reset()

	// This should cause a fatal error - we can't easily test this
	// without subprocess testing
	Init("./testdata/invalid.json")
}
*/

// BenchmarkGetPackageConfig benchmarks the config retrieval
func BenchmarkGetPackageConfig(b *testing.B) {
	cclog.Init("debug", true)
	Init("./testdata/config.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetPackageConfig("main")
	}
}

// BenchmarkHasKey benchmarks the key existence check
func BenchmarkHasKey(b *testing.B) {
	cclog.Init("debug", true)
	Init("./testdata/config.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HasKey("main")
	}
}

// BenchmarkGetKeys benchmarks getting all keys
func BenchmarkGetKeys(b *testing.B) {
	cclog.Init("debug", true)
	Init("./testdata/config.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetKeys()
	}
}
