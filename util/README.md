# Util Package

The `util` package provides a collection of utility functions and types for common operations in the ClusterCockpit library.

## Overview

This package contains utilities for:
- **Array operations** - Generic helper functions for slices
- **File compression** - Gzip compression and decompression
- **File/directory operations** - Copying files and directories
- **Disk usage** - Calculating directory size
- **Custom types** - Selector types
- **File system watcher** - Event-based file system monitoring
- **Statistics** - Basic statistical functions (mean, median, min, max)
- **Secrets** - Resolving secrets from environment variables or secret files

The `Float` type with JSON NaN support is in the [`schema`](../schema) package,
not here.

## Key Features

### Secrets from the Environment

A secret should not have to live in a configuration file. `SecretFromEnv`
resolves one from three sources, in this order of precedence:

1. the environment variable `$VAR`, if set and non-empty
2. the contents of the file named by `$VAR_FILE`
3. the value read from the configuration file

```go
// Resolves $NATS_PASSWORD, else the file named by $NATS_PASSWORD_FILE,
// else the value from the config file.
password, err := util.SecretFromEnv("CC_NATS_PASSWORD", cfg.Password)
if err != nil {
    return err
}
```

The second step is what makes the standard secret-mounting mechanisms usable:

```bash
# Docker / docker compose
docker run -e CC_NATS_PASSWORD_FILE=/run/secrets/nats-pw ...

# Kubernetes: mount the secret, then name the path
#   env:
#     - name: CC_NATS_PASSWORD_FILE
#       value: /etc/secrets/nats-pw

# systemd
[Service]
LoadCredential=nats-pw:/etc/cc/nats-pw
Environment=CC_NATS_PASSWORD_FILE=%d/nats-pw
```

For a repeated configuration section — one sink, one receiver — no fixed
variable name can address a single instance, so the instance names its own
sources through sibling configuration keys and resolves them with
`SecretFromConfig`:

```go
// {"password": "...", "password_env": "...", "password_file": "..."}
password, err := util.SecretFromConfig(cfg.Password, cfg.PasswordEnv, cfg.PasswordFile)
```

Rules that apply to both functions:

- An empty environment variable counts as unset.
- File contents are trimmed of surrounding whitespace, so a secret file may end
  in a newline. A secret with significant leading or trailing whitespace cannot
  be supplied through a file.
- A secret file that cannot be read, or that holds no non-whitespace
  characters, is an **error** rather than a silent fallback to the configured
  value. An operator who names a file intends it to win, so falling back would
  quietly start the process with a stale credential.
- Neither function ever logs a resolved secret. Errors carry only the path.

Environment variables that cc-lib itself reads are prefixed `CC_`, so they
cannot collide with a variable already present in the environment of the
application linking cc-lib.

### File Operations

```go
// Compress a file
err := util.CompressFile("input.txt", "output.txt.gz")

// Decompress a file
err = util.UncompressFile("input.txt.gz", "output.txt")

// Copy a file
err = util.CopyFile("source.txt", "destination.txt")

// Copy a directory recursively
err = util.CopyDir("/path/to/source", "/path/to/dest")
```

### Disk Usage

```go
// Get disk usage in megabytes for a directory
usage := util.DiskUsage("/path/to/directory")
fmt.Printf("Directory uses %.2f MB\n", usage)
```

### Array Utilities

```go
// Check if a slice contains an element (works with any comparable type)
numbers := []int{1, 2, 3, 4, 5}
contains := util.Contains(numbers, 3) // true

strs := []string{"apple", "banana", "orange"}
contains = util.Contains(strs, "grape") // false
```

### Statistics

```go
data := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

// Calculate mean
mean, err := util.Mean(data) // 3.0

// Calculate median
median, err := util.Median(data) // 3.0

// Min/Max (works with any ordered type)
minVal := util.Min(5, 3) // 3
maxVal := util.Max(5, 3) // 5
```

### File System Watcher

```go
// Implement the Listener interface
type MyListener struct{}

func (l *MyListener) EventCallback() {
    fmt.Println("File changed!")
}

func (l *MyListener) EventMatch(event string) bool {
    return strings.Contains(event, "myfile.txt")
}

// Add a listener
listener := &MyListener{}
util.AddListener("/path/to/watch", listener)

// Don't forget to shutdown when done
defer util.FsWatcherShutdown()
```

### Selector Types

The `SelectorElement` and `Selector` types support flexible JSON marshaling for configuration:

```go
// Can be a single string
var sel util.SelectorElement
json.Unmarshal([]byte(`"value"`), &sel)

// Can be an array of strings
json.Unmarshal([]byte(`["val1", "val2"]`), &sel)

// Can be a wildcard
json.Unmarshal([]byte(`"*"`), &sel)
```

## Documentation

For complete API documentation, see the [godoc](https://pkg.go.dev/github.com/ClusterCockpit/cc-lib/v2/util).

## Testing

The package includes comprehensive unit tests. Run them with:

```bash
go test ./util/...
```

For coverage information:

```bash
go test -cover ./util/...
```
