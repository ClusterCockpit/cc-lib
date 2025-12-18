<!--
---
title: ClusterCockpit logger
description: Complete guide to the ClusterCockpit logging interface
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccLogger/_index.md
---
-->

# ccLogger Interface

The ccLogger component provides a unified logging interface for ClusterCockpit applications. It wraps Go's standard `log` package with support for multiple log levels and integrates seamlessly with systemd's journaling system.

## Features

- **Multiple log levels**: debug, info, warn, error, critical
- **Systemd integration**: Uses systemd priority prefixes for proper log categorization
- **Flexible output**: Log to stderr (default), files, or custom writers
- **Thread-safe**: All logging functions are safe for concurrent use
- **Component tagging**: Built-in support for component-specific logging
- **Formatted and unformatted logging**: Choose between simple or printf-style logging

## Quick Start

```go
import "github.com/ClusterCockpit/cc-lib/ccLogger"

func main() {
    // Initialize with log level and timestamp preference
    cclogger.Init("info", false) // info level, no timestamps (systemd adds them)
    
    // Log messages
    cclogger.Info("Application started")
    cclogger.Warnf("Configuration file %s not found, using defaults", configPath)
    cclogger.Error("Failed to connect to database")
}
```

## Log Levels

Log levels in order of increasing severity:

| Level | Use Case | Example |
|-------|----------|---------|
| **debug** | Detailed development/troubleshooting information | `cclogger.Debug("Processing item 42")` |
| **info** | General informational messages | `cclogger.Info("Server started on port 8080")` |
| **warn** | Important but non-critical issues | `cclogger.Warn("Cache miss, fetching from database")` |
| **err**/**fatal** | Errors that allow continued execution | `cclogger.Error("Failed to send email notification")` |
| **crit** | Critical errors leading to termination | `cclogger.Fatal("Cannot bind to port 8080")` (exits) |

### Setting Log Level

When you initialize cclogger with a specific level, only messages at that level and above are displayed:

```go
cclogger.Init("warn", false)
// Now only warn, error, and critical messages are shown
// Debug and info messages are suppressed
```

## Basic Usage

### Simple Logging

```go
cclogger.Debug("Detailed debug information")
cclogger.Info("User logged in successfully")
cclogger.Warn("Retry attempt 3 of 5")
cclogger.Error("Database query failed")
cclogger.Fatal("Critical error, exiting") // Exits with code 1
```

### Formatted Logging

Use `printf`-style formatting for structured output:

```go
cclogger.Debugf("Processing item %d of %d", current, total)
cclogger.Infof("User %s logged in from %s", username, ipAddress)
cclogger.Warnf("Cache size %d MB exceeds threshold %d MB", size, threshold)
cclogger.Errorf("Failed to open file %s: %v", filename, err)
```

### Component Logging

Tag log messages with component names for better organization:

```go
cclogger.ComponentInfo("scheduler", "Job queue initialized")
cclogger.ComponentError("database", "Connection pool exhausted")
cclogger.ComponentWarn("auth", "Failed login attempt from", ipAddr)
cclogger.ComponentDebug("cache", "Cache hit rate:", hitRate)
```

## Advanced Usage

### Logging to Files

Redirect logs to a file for specific levels:

```go
// Log warn, info, and debug to file
// Error and critical still go to stderr
cclogger.SetOutputFile("warn", "/var/log/myapp.log")
```

**Note**: The file remains open for the lifetime of the application. This is intentional to allow continuous logging.

### Timestamps

Control timestamp inclusion based on your environment:

```go
// No timestamps (recommended for systemd)
cclogger.Init("info", false)

// With timestamps (for traditional logging)
cclogger.Init("info", true)
```

### Custom Writers

You can customize output destinations by modifying the writers before initialization:

```go
import "os"

// Direct debug logs to a custom writer
cclogger.DebugWriter = myCustomWriter
cclogger.Init("debug", false)
```

## Systemd Integration

ccLogger uses systemd's priority prefixes to enable proper log categorization in journald:

- `<7>` - Debug (LOG_DEBUG)
- `<6>` - Info (LOG_INFO)
- `<4>` - Warning (LOG_WARNING)
- `<3>` - Error (LOG_ERR)
- `<2>` - Critical (LOG_CRIT)

When running under systemd, these prefixes allow journald to automatically:
- Filter logs by priority
- Add metadata (timestamps, service name, etc.)
- Enable structured querying with `journalctl`

Example journalctl usage:
```bash
# View only warning and above
journalctl -u myservice -p warning

# View logs from specific component (if using ComponentInfo etc.)
journalctl -u myservice | grep '\[scheduler\]'
```

## Best Practices

1. **Choose the right level**:
   - Use `Debug` for verbose diagnostic information
   - Use `Info` for normal application flow events
   - Use `Warn` for unusual but handled situations
   - Use `Error` for failures that don't stop the application
   - Use `Fatal` only for unrecoverable errors

2. **Disable timestamps for systemd**: When running as a systemd service, use `Init(level, false)` to avoid duplicate timestamps

3. **Use component logging**: For multi-component applications, use `Component*` functions to make log filtering easier

4. **Structured logging**: Use formatted variants (`*f` functions) to create parseable log messages:
   ```go
   cclogger.Infof("user=%s action=%s status=%s", user, action, status)
   ```

5. **Thread safety**: All logging functions are thread-safe, so you can safely log from goroutines

6. **Error context**: Always include relevant context in error messages:
   ```go
   cclogger.Errorf("Failed to connect to %s:%d: %v", host, port, err)
   ```

## API Reference

For complete API documentation, see the [package documentation](https://pkg.go.dev/github.com/ClusterCockpit/cc-lib/ccLogger) or run:

```bash
go doc github.com/ClusterCockpit/cc-lib/ccLogger
```

## Examples

### Basic Application

```go
package main

import "github.com/ClusterCockpit/cc-lib/ccLogger"

func main() {
    cclogger.Init("info", false)
    
    cclogger.Info("Application starting")
    
    if err := doSomething(); err != nil {
        cclogger.Errorf("Operation failed: %v", err)
        return
    }
    
    cclogger.Info("Application completed successfully")
}
```

### Multi-Component Service

```go
func startScheduler() {
    cclogger.ComponentInfo("scheduler", "Starting job scheduler")
    
    for job := range jobQueue {
        cclogger.ComponentDebug("scheduler", "Processing job", job.ID)
        
        if err := processJob(job); err != nil {
            cclogger.ComponentError("scheduler", "Job failed:", job.ID, err)
        }
    }
}

func connectDatabase() error {
    cclogger.ComponentInfo("database", "Connecting to database")
    
    if err := db.Connect(); err != nil {
        cclogger.ComponentError("database", "Connection failed:", err)
        return err
    }
    
    cclogger.ComponentInfo("database", "Connected successfully")
    return nil
}
```
