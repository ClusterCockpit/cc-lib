<!--
---
title: ClusterCockpit configuration component
description: Description of ClusterCockpit's configuration component
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccConfig/_index.md
---
-->

# ccConfig Interface

The `ccConfig` package provides a simple yet flexible configuration management system for the ClusterCockpit ecosystem. It allows components to load JSON configuration files with support for both inline configurations and external file references.

## Features

- **Simple API**: Just two main functions - `Init()` to load and `GetPackageConfig()` to retrieve
- **File References**: Support for splitting configuration across multiple files
- **Flexible Structure**: Organize config by component/package
- **Helper Functions**: Check for existence, list keys, and reset configuration
- **Zero Dependencies**: Uses only standard library (plus ccLogger)

## Configuration File Structure

All configuration files follow this JSON structure:

```json
{
    "main": {
        "interval": "10s",
        "debug": true
    },
    "database": {
        "host": "localhost",
        "port": 5432
    },
    "sinks-file": "./sinks.json"
}
```

Each top-level key represents a configuration section that can be retrieved independently.

## File References

Keys ending with `-file` are treated specially. The value should be a string path to an external JSON file:

```json
{
    "receivers-file": "./receivers.json",
    "sinks-file": "./sinks.json",
    "main": {
        "interval": "10s"
    }
}
```

In the example above:
- `receivers.json` content will be available via `GetPackageConfig("receivers")`
- `sinks.json` content will be available via `GetPackageConfig("sinks")`
- The inline `main` config is available via `GetPackageConfig("main")`

## API Reference

### Init(filename string)

Initializes the configuration system by loading a JSON configuration file.

```go
ccconfig.Init("./config.json")
```

**Behavior:**
- If the file doesn't exist, continues with empty configuration
- If the file is invalid JSON, terminates with fatal error
- Processes all `-file` references and loads external files

### GetPackageConfig(key string) json.RawMessage

Retrieves raw JSON configuration for a specific key.

```go
rawConfig := ccconfig.GetPackageConfig("database")
if rawConfig == nil {
    log.Fatal("database config not found")
}

var dbConfig DatabaseConfig
if err := json.Unmarshal(rawConfig, &dbConfig); err != nil {
    log.Fatalf("invalid config: %v", err)
}
```

**Returns:**
- `json.RawMessage` containing the configuration data
- `nil` if the key doesn't exist (also logs an info message)

### HasKey(key string) bool

Checks whether a configuration key exists.

```go
if ccconfig.HasKey("optional-metrics") {
    config := ccconfig.GetPackageConfig("optional-metrics")
    // ... use config
}
```

### GetKeys() []string

Returns all available configuration keys.

```go
keys := ccconfig.GetKeys()
for _, key := range keys {
    fmt.Printf("Config section: %s\n", key)
}
```

### Reset()

Clears all loaded configuration. Useful for testing.

```go
ccconfig.Reset()
ccconfig.Init("test-config.json")
```

## Usage Example

```go
package main

import (
    "encoding/json"
    "log"
    ccconfig "github.com/ClusterCockpit/cc-lib/ccConfig"
)

type AppConfig struct {
    Interval string `json:"interval"`
    Debug    bool   `json:"debug"`
    Workers  int    `json:"workers"`
}

func main() {
    // Initialize configuration
    ccconfig.Init("./config.json")

    // Check if configuration exists
    if !ccconfig.HasKey("myapp") {
        log.Fatal("myapp configuration section not found")
    }

    // Retrieve and parse configuration
    rawConfig := ccconfig.GetPackageConfig("myapp")
    var config AppConfig
    if err := json.Unmarshal(rawConfig, &config); err != nil {
        log.Fatalf("failed to parse config: %v", err)
    }

    // Use configuration
    log.Printf("Starting with %d workers, interval: %s", 
               config.Workers, config.Interval)
}
```

## Best Practices

### 1. Always Check for nil

```go
rawConfig := ccconfig.GetPackageConfig("component")
if rawConfig == nil {
    // Handle missing configuration appropriately
    log.Fatal("required configuration not found")
}
```

### 2. Use Struct Tags for JSON

```go
type Config struct {
    Interval string `json:"interval"`
    Enabled  bool   `json:"enabled"`
    // Use omitempty for optional fields
    OptionalField string `json:"optional_field,omitempty"`
}
```

### 3. Organize Configuration by Component

Split large configurations into separate files:

```json
{
    "main": { ... },
    "receivers-file": "./receivers.json",
    "sinks-file": "./sinks.json",
    "collectors-file": "./collectors.json"
}
```

### 4. Validate After Loading

```go
var config AppConfig
if err := json.Unmarshal(rawConfig, &config); err != nil {
    log.Fatalf("invalid config: %v", err)
}

// Validate configuration values
if config.Interval == "" {
    log.Fatal("interval must be specified")
}
if config.Workers < 1 {
    log.Fatal("workers must be at least 1")
}
```

### 5. Use Reset() in Tests

```go
func TestMyComponent(t *testing.T) {
    ccconfig.Reset() // Clear any previous config
    ccconfig.Init("./testdata/test-config.json")
    
    // ... your tests
}
```

## Troubleshooting

### Configuration Not Found

**Problem:** `GetPackageConfig()` returns `nil`

**Solutions:**
1. Check that `Init()` was called with the correct file path
2. Verify the key name matches exactly (case-sensitive)
3. For file references, ensure the `-file` suffix is in the config, not in your retrieval key
4. Use `GetKeys()` to see all available keys

```go
// Debug: Print all available keys
keys := ccconfig.GetKeys()
fmt.Printf("Available keys: %v\n", keys)
```

### Invalid JSON Error

**Problem:** Fatal error during `Init()` about invalid JSON

**Solutions:**
1. Validate your JSON with a linter (e.g., `jq . < config.json`)
2. Check for trailing commas (not allowed in JSON)
3. Ensure all strings use double quotes, not single quotes
4. Verify file encoding is UTF-8

### File Reference Not Loading

**Problem:** External file referenced via `-file` suffix not loading

**Solutions:**
1. Check file path is relative to the current working directory
2. Verify file exists and is readable
3. Ensure the referenced file contains valid JSON
4. Check logs for error messages from ccLogger

### Unmarshal Errors

**Problem:** `json.Unmarshal()` fails when parsing retrieved config

**Solutions:**
1. Ensure struct field types match JSON types
2. Use JSON tags that match your config keys exactly
3. Make sure struct fields are exported (start with capital letter)
4. Enable debug logging to see the raw JSON being parsed

```go
rawConfig := ccconfig.GetPackageConfig("myapp")
fmt.Printf("Raw config: %s\n", string(rawConfig)) // Debug print
```

## Implementation Notes

- Configuration is stored in a global map after initialization
- The package does not support hot-reloading (requires restart to pick up changes)
- File references are resolved at initialization time
- Missing config files (os.IsNotExist) are silently ignored, other errors are fatal
- The package is **not thread-safe** - ensure `Init()` is called before concurrent access
