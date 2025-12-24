// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package ccconfig provides a simple configuration management system for the ClusterCockpit ecosystem.
//
// The package supports loading JSON configuration files with a flexible structure that allows
// both inline configuration and external file references. Configuration sections can be accessed
// by key, enabling modular configuration management across different components.
//
// # Configuration File Structure
//
// Configuration files should follow this JSON structure:
//
//	{
//	    "main": {
//	        // Main configuration for the component
//	    },
//	    "foo": {
//	        // Configuration for sub-component 'foo'
//	    },
//	    "bar-file": "path/to/bar.json"
//	}
//
// # File References
//
// Keys ending with "-file" are treated as file references. The value should be a string
// path to an external JSON file. The referenced file's content will be loaded and stored
// under the key prefix (without "-file" suffix).
//
// For example, "sinks-file": "sinks.json" will load the content of sinks.json
// and make it available via GetPackageConfig("sinks").
//
// # Usage Example
//
//	package main
//
//	import (
//	    "encoding/json"
//	    "log"
//	    ccconfig "github.com/ClusterCockpit/cc-lib/v2/ccConfig"
//	)
//
//	type AppConfig struct {
//	    Interval string `json:"interval"`
//	    Debug    bool   `json:"debug"`
//	}
//
//	func main() {
//	    // Initialize with config file
//	    ccconfig.Init("config.json")
//
//	    // Get configuration for your package
//	    rawConfig := ccconfig.GetPackageConfig("myapp")
//	    if rawConfig == nil {
//	        log.Fatal("myapp configuration not found")
//	    }
//
//	    // Unmarshal into your config struct
//	    var config AppConfig
//	    if err := json.Unmarshal(rawConfig, &config); err != nil {
//	        log.Fatalf("failed to parse config: %v", err)
//	    }
//
//	    // Use the configuration
//	    log.Printf("Running with interval: %s", config.Interval)
//	}
package ccconfig

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// keys holds the parsed configuration data indexed by key name.
// File references (keys ending with "-file") are resolved and stored
// under their base key name (without the "-file" suffix).
var keys map[string]json.RawMessage

// Init initializes the configuration system by loading and parsing a JSON configuration file.
// It reads the specified file and processes all configuration sections. Keys ending with "-file"
// are treated as references to external files, which are loaded and stored under the base key name.
//
// If the file does not exist, Init will silently continue with an empty configuration.
// Other errors (permission denied, invalid JSON, etc.) will cause the program to terminate
// with a fatal error message.
//
// Parameters:
//   - filename: Path to the main JSON configuration file
//
// Example:
//
//	ccconfig.Init("./config.json")
func Init(filename string) {
	raw, err := os.ReadFile(filename)
	jkeys := make(map[string]json.RawMessage)

	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("CONFIG ERROR: %v", err)
		}
	} else {
		dec := json.NewDecoder(bytes.NewReader(raw))
		if err := dec.Decode(&jkeys); err != nil {
			log.Fatalf("could not decode: %v", err)
		}
	}

	keys = make(map[string]json.RawMessage)

	// Process each key-value pair, handling file references
	for k, v := range jkeys {
		s := strings.Split(k, "-")
		// Check if this is a file reference (key ends with "-file")
		if len(s) == 2 && s[1] == "file" {
			var filename string
			err := json.Unmarshal(v, &filename)
			if err != nil {
				log.Fatalln("error:", err)
			}
			b, err := os.ReadFile(filename)
			if err != nil {
				cclog.ComponentError("ccConfig", err.Error())
			}

			keys[s[0]] = b
		} else {
			keys[k] = jkeys[k]
		}
	}
}

// GetPackageConfig retrieves the raw JSON configuration for a given key.
// It returns the configuration as json.RawMessage which can then be unmarshaled
// into the appropriate structure.
//
// If the key does not exist, it logs an informational message and returns nil.
// Callers should check for nil before attempting to unmarshal the result.
//
// Parameters:
//   - key: The configuration key to retrieve
//
// Returns:
//   - json.RawMessage containing the configuration data, or nil if the key doesn't exist
//
// Example:
//
//	rawConfig := ccconfig.GetPackageConfig("database")
//	if rawConfig != nil {
//	    var dbConfig DatabaseConfig
//	    json.Unmarshal(rawConfig, &dbConfig)
//	}
func GetPackageConfig(key string) json.RawMessage {
	if val, ok := keys[key]; ok {
		return val
	}
	cclog.Infof("CONFIG INFO: Key %s not found", key)
	return nil
}

// Reset clears all loaded configuration data.
// This is primarily useful for testing scenarios where you need to reload
// or clear configuration between test cases.
//
// Example:
//
//	ccconfig.Reset()
//	ccconfig.Init("test-config.json")
func Reset() {
	keys = make(map[string]json.RawMessage)
}

// HasKey checks whether a configuration key exists.
// This can be used to determine if a component's configuration was loaded
// before attempting to retrieve it.
//
// Parameters:
//   - key: The configuration key to check
//
// Returns:
//   - true if the key exists, false otherwise
//
// Example:
//
//	if ccconfig.HasKey("optional-feature") {
//	    config := ccconfig.GetPackageConfig("optional-feature")
//	    // ... use config
//	}
func HasKey(key string) bool {
	_, ok := keys[key]
	return ok
}

// GetKeys returns a list of all available configuration keys.
// This can be useful for debugging or for components that need to discover
// what configuration sections are available.
//
// Returns:
//   - A slice of strings containing all configuration keys
//
// Example:
//
//	keys := ccconfig.GetKeys()
//	for _, key := range keys {
//	    fmt.Printf("Found config section: %s\n", key)
//	}
func GetKeys() []string {
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}
