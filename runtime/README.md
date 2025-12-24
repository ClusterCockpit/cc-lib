<!--
---
title: Runtime Environment Utilities
description: Package for environment setup, privilege dropping, and systemd integration
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/runtimeEnv/_index.md
---
-->

# runtimeEnv

The `runtimeEnv` package provides utilities for runtime environment setup and systemd integration for ClusterCockpit applications. It enables secure privilege management, environment configuration, and proper systemd service lifecycle integration.

## Features

- **Environment file loading**: Read and parse .env configuration files
- **Privilege dropping**: Securely drop from root to unprivileged users
- **Systemd integration**: Service readiness notifications and status updates
- **Thread-safe**: All functions safe for concurrent use
- **Cross-platform**: Works on Linux systems (privilege dropping is Linux-specific)

## Installation

```go
import "github.com/ClusterCockpit/cc-lib/v2/runtimeEnv"
```

## Quick Start

```go
package main

import (
    "log"
    "os"
    
    "github.com/ClusterCockpit/cc-lib/v2/runtimeEnv"
)

func main() {
    // Load optional .env file
    if err := runtimeEnv.LoadEnv("./.env"); err != nil && !os.IsNotExist(err) {
        log.Fatalf("Failed to load .env: %v", err)
    }
    
    // Start server (may require root for port < 1024)
    if err := startServer(":80"); err != nil {
        log.Fatal(err)
    }
    
    // Drop privileges for security
    if err := runtimeEnv.DropPrivileges("www-data", "www-data"); err != nil {
        log.Fatal(err)
    }
    
    // Notify systemd we're ready
    runtimeEnv.SystemdNotify(true, "Running")
    
    // Serve requests
    serve()
}
```

## Functions

### LoadEnv

Load environment variables from a .env file.

```go
func LoadEnv(file string) error
```

**Supported .env syntax:**

```bash
# Comments (must be at start of line)
SIMPLE_VAR=value
export EXPORTED_VAR=value
QUOTED_VAR="value with spaces"
ESCAPED_VAR="line1\nline2\ttabbed"
```

**Escape sequences in quoted strings:**
- `\n` - newline
- `\r` - carriage return
- `\t` - tab
- `\"` - double quote

**Limitations:**
- Comments only allowed at line start (not inline)
- Only double quotes supported
- No variable expansion/substitution
- No multi-line values

**Example:**

```go
// Load required .env file
if err := runtimeEnv.LoadEnv("config.env"); err != nil {
    log.Fatal(err)
}

// Load optional .env file
if err := runtimeEnv.LoadEnv(".env"); err != nil && !os.IsNotExist(err) {
    log.Fatalf("Failed to load .env: %v", err)
}

// Now use environment variables
dbHost := os.Getenv("DB_HOST")
```

**Sample .env file:**

```bash
# Database configuration
DB_HOST=localhost
DB_PORT=5432
export DB_NAME=clustercockpit
DB_PASSWORD="secret password with spaces"

# Logging
LOG_LEVEL=info
LOG_FORMAT="[%level%]\t%message%\n"
```

### DropPrivileges

Permanently drop root privileges to an unprivileged user.

```go
func DropPrivileges(username string, group string) error
```

**Security best practices:**

1. **Drop early**: Call as soon as privileged operations complete
2. **Verify user exists**: Ensure user/group exist before starting
3. **Irreversible**: Cannot regain root privileges after calling
4. **Both or user only**: Can drop both user+group or just user

**Parameters:**
- `username` - Username to switch to (empty string skips)
- `group` - Group name to switch to (empty string skips)

**Example 1: Basic usage**

```go
// Drop to dedicated service user
if err := runtimeEnv.DropPrivileges("ccuser", "ccgroup"); err != nil {
    log.Fatalf("Failed to drop privileges: %v", err)
}
```

**Example 2: Only change user**

```go
// Keep current group
if err := runtimeEnv.DropPrivileges("nobody", ""); err != nil {
    log.Fatal(err)
}
```

**Example 3: Typical server pattern**

```go
func main() {
    // Bind to privileged port (requires root)
    listener, err := net.Listen("tcp", ":80")
    if err != nil {
        log.Fatal(err)
    }
    
    // Drop privileges before handling requests
    if err := runtimeEnv.DropPrivileges("www-data", "www-data"); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Now running as www-data user")
    
    // Serve requests as unprivileged user
    http.Serve(listener, handler)
}
```

**Example 4: Conditional privilege dropping**

```go
func main() {
    // Only drop if running as root
    if os.Geteuid() == 0 {
        log.Println("Running as root, dropping privileges")
        if err := runtimeEnv.DropPrivileges("ccuser", "ccgroup"); err != nil {
            log.Fatal(err)
        }
    } else {
        log.Println("Not running as root, keeping current user")
    }
}
```

### SystemdNotify

Send status notifications to systemd.

```go
func SystemdNotify(ready bool, status string)
```

**Parameters:**
- `ready` - If true, signals service is ready (sends --ready)
- `status` - Status message for systemctl status (optional)

**Behavior:**
- Safe to call in non-systemd environments (checks NOTIFY_SOCKET)
- Errors are ignored (service continues running)
- Does nothing if not running under systemd

**Example 1: Signal readiness**

```go
// After initialization completes
runtimeEnv.SystemdNotify(true, "Ready to accept connections")
```

**Example 2: Status updates**

```go
// Update status without signaling ready
runtimeEnv.SystemdNotify(false, "Processing 1000 requests/sec")
```

**Example 3: Shutdown notification**

```go
func main() {
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    // Start service
    go serve()
    runtimeEnv.SystemdNotify(true, "Running")
    
    // Wait for shutdown signal
    <-sigChan
    runtimeEnv.SystemdNotify(false, "Shutting down gracefully")
    
    // Cleanup
    cleanup()
}
```

**Example 4: Complete service lifecycle**

```go
func main() {
    log.Println("Initializing...")
    if err := initialize(); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Starting server...")
    if err := startServer(); err != nil {
        log.Fatal(err)
    }
    
    // Signal systemd we're ready
    runtimeEnv.SystemdNotify(true, "Running")
    log.Println("Service ready")
    
    // Update status periodically
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for range ticker.C {
            stats := getStats()
            runtimeEnv.SystemdNotify(false, 
                fmt.Sprintf("Active connections: %d", stats.Connections))
        }
    }()
    
    // Run service
    serve()
}
```

## Systemd Service Configuration

**Basic service file:**

```ini
[Unit]
Description=ClusterCockpit Service
After=network.target

[Service]
Type=notify
User=ccuser
Group=ccgroup
ExecStart=/usr/bin/myservice
NotifyAccess=main
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

**With environment file:**

```ini
[Service]
Type=notify
EnvironmentFile=/etc/myservice/service.env
ExecStart=/usr/bin/myservice
NotifyAccess=main
```

## Complete Examples

### Example 1: ClusterCockpit Collector

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
    "github.com/ClusterCockpit/cc-lib/v2/runtimeEnv"
)

func main() {
    // Load optional config
    _ = runtimeEnv.LoadEnv("./.env")
    
    // Initialize logger
    ccLogger.Init(os.Getenv("LOG_LEVEL"), false)
    
    // Initialize collector
    ccLogger.Info("Initializing collector")
    if err := initCollector(); err != nil {
        ccLogger.Fatal(err)
    }
    
    // Drop privileges if running as root
    if os.Geteuid() == 0 {
        user := os.Getenv("RUN_USER")
        group := os.Getenv("RUN_GROUP")
        if user == "" {
            user = "nobody"
        }
        if group == "" {
            group = "nogroup"
        }
        
        if err := runtimeEnv.DropPrivileges(user, group); err != nil {
            ccLogger.Fatalf("Failed to drop privileges: %v", err)
        }
        ccLogger.Infof("Dropped privileges to %s:%s", user, group)
    }
    
    // Start collection
    ccLogger.Info("Starting metric collection")
    go collect()
    
    // Signal systemd
    runtimeEnv.SystemdNotify(true, "Collecting metrics")
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    <-sigChan
    
    runtimeEnv.SystemdNotify(false, "Shutting down")
    ccLogger.Info("Shutdown complete")
}
```

### Example 2: Web Server with Privilege Dropping

```go
package main

import (
    "log"
    "net/http"
    "os"
    
    "github.com/ClusterCockpit/cc-lib/v2/runtimeEnv"
)

func main() {
    // Load config
    if err := runtimeEnv.LoadEnv("server.env"); err != nil {
        log.Fatal(err)
    }
    
    // Create listener on privileged port (requires root)
    port := os.Getenv("PORT")
    if port == "" {
        port = "80"
    }
    
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("Failed to bind to port %s: %v", port, err)
    }
    
    // Drop to unprivileged user
    if err := runtimeEnv.DropPrivileges("www-data", "www-data"); err != nil {
        log.Fatal(err)
    }
    log.Println("Privileges dropped to www-data")
    
    // Setup routes
    http.HandleFunc("/", handleRequest)
    
    // Notify systemd
    runtimeEnv.SystemdNotify(true, "Serving HTTP on :"+port)
    
    // Serve (already have listener from root)
    log.Fatal(http.Serve(listener, nil))
}
```

## Error Handling

All functions return errors that should be checked:

```go
// LoadEnv - handle file not found separately
if err := runtimeEnv.LoadEnv(".env"); err != nil {
    if os.IsNotExist(err) {
        log.Println("No .env file, using defaults")
    } else {
        log.Fatalf("Error loading .env: %v", err)
    }
}

// DropPrivileges - always fatal
if err := runtimeEnv.DropPrivileges("user", "group"); err != nil {
    log.Fatalf("Cannot drop privileges: %v", err)
}

// SystemdNotify - no return value, errors ignored internally
runtimeEnv.SystemdNotify(true, "Running")
```

## Thread Safety

All functions are thread-safe and can be called from multiple goroutines. However:

- **LoadEnv**: Safe to call concurrently, but typically called once at startup
- **DropPrivileges**: Should only be called **once** during initialization
- **SystemdNotify**: Safe to call frequently from multiple goroutines

## Platform Notes

- **LoadEnv**: Works on all platforms
- **DropPrivileges**: Linux only (uses syscall.Setuid/Setgid)
- **SystemdNotify**: Linux only (requires systemd), safe no-op on other platforms

## Testing

The package includes comprehensive tests for all functions. Run tests with:

```bash
go test -v github.com/ClusterCockpit/cc-lib/v2/runtimeEnv
```

## Security Considerations

1. **Privilege Dropping**:
   - Always drop privileges as early as possible
   - Verify user/group exist before starting service
   - Test your service runs correctly as unprivileged user
   - Never try to regain privileges after dropping

2. **Environment Files**:
   - Protect .env files with appropriate permissions (0600 or 0640)
   - Never commit .env files with secrets to version control
   - Use .env.example for templates without secrets

3. **Best Practices**:
   - Use dedicated service users (not nobody/nogroup in production)
   - Run with minimal filesystem access
   - Use systemd's additional security features (PrivateTmp, NoNewPrivileges, etc.)

## API Reference

For complete API documentation, see [pkg.go.dev](https://pkg.go.dev/github.com/ClusterCockpit/cc-lib/v2/runtimeEnv).

## License

Copyright (C) NHR@FAU, University Erlangen-Nuremberg.  
Licensed under the MIT License. See LICENSE file for details.

## See Also

- [ccLogger](../ccLogger/README.md) - Logging with systemd integration
- [ccConfig](../ccConfig/README.md) - Configuration management
- [systemd sd_notify](https://www.freedesktop.org/software/systemd/man/sd_notify.html) - Systemd notification protocol
