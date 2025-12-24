# Util Package

The `util` package provides a collection of utility functions and types for common operations in the ClusterCockpit library.

## Overview

This package contains utilities for:
- **Array operations** - Generic helper functions for slices
- **File compression** - Gzip compression and decompression
- **File/directory operations** - Copying files and directories
- **Disk usage** - Calculating directory size
- **Custom types** - Float type with JSON NaN support, Selector types
- **File system watcher** - Event-based file system monitoring
- **Statistics** - Basic statistical functions (mean, median, min, max)

## Key Features

### Float Type with NaN Support

Go's standard JSON encoder doesn't support NaN values (see [golang/go#3480](https://github.com/golang/go/issues/3480)). This package provides a `Float` type that properly handles NaN values in JSON by converting them to/from `null`.

```go
import "github.com/ClusterCockpit/cc-lib/v2/util"

// Create a Float value
f := util.Float(3.14)

// Use NaN to represent missing data
missing := util.NaN

// JSON marshaling - NaN becomes null
data, _ := json.Marshal(missing) // Returns: null
```

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
