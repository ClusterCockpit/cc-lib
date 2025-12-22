<!--
---
title: ClusterCockpit messages
description: Description of ClusterCockpit message format and interface
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/ccMessage/_index.md
---
-->

# ClusterCockpit messages

As described in the [ClusterCockpit specifications](https://github.com/ClusterCockpit/cc-specifications), the whole ClusterCockpit stack uses metrics, events and control in the InfluxDB line protocol format. This is also the input and output format for the ClusterCockpit Metric Collector but internally it uses an extended format while processing, named CCMessage.

It is basically a copy of the [InfluxDB line protocol](https://github.com/influxdata/line-protocol) `MutableMetric` interface with one extension. Besides the tags and fields, it contains a list of meta information (re-using the `Tag` structure of the original protocol):

```golang
type ccMessage struct {
    name   string                 // Measurement name
    meta   map[string]string      // map of meta data tags
    tags   map[string]string      // map of of tags
    fields map[string]interface{} // map of of fields
    tm     time.Time              // timestamp
}

type CCMessage interface {
    ToPoint(metaAsTags map[string]bool) *write.Point  // Generate influxDB point for data type ccMessage
    ToLineProtocol(metaAsTags map[string]bool) string // Generate influxDB line protocol for data type ccMessage
    String() string                                   // Return line-protocol like string

    Name() string        // Get metric name
    SetName(name string) // Set metric name

    Time() time.Time     // Get timestamp
    SetTime(t time.Time) // Set timestamp

    Tags() map[string]string                   // Map of tags
    AddTag(key, value string)                  // Add a tag
    GetTag(key string) (value string, ok bool) // Get a tag by its key
    HasTag(key string) (ok bool)               // Check if a tag key is present
    RemoveTag(key string)                      // Remove a tag by its key

    Meta() map[string]string                    // Map of meta data tags
    AddMeta(key, value string)                  // Add a meta data tag
    GetMeta(key string) (value string, ok bool) // Get a meta data tab addressed by its key
    HasMeta(key string) (ok bool)               // Check if a meta data key is present
    RemoveMeta(key string)                      // Remove a meta data tag by its key

    Fields() map[string]interface{}                   // Map of fields
    AddField(key string, value interface{})           // Add a field
    GetField(key string) (value interface{}, ok bool) // Get a field addressed by its key
    HasField(key string) (ok bool)                    // Check if a field key is present
    RemoveField(key string)                           // Remove a field addressed by its key
}

func NewMessage(name string, tags map[string]string, meta map[string]string, fields map[string]interface{}, tm time.Time) (CCMessage, error)
func FromMessage(other CCMessage) CCMessage
func FromBytes(data []byte) ([]CCMessage, error)
```

The `CCMessage` interface provides the same functions as the `MutableMetric` like `{Add, Get, Remove, Has}{Tag, Field}` and additionally provides `{Add, Get, Remove, Has}Meta`.

The InfluxDB protocol creates a new metric with `influx.New(name, tags, fields, time)` while CCMessage uses `ccMessage.New(name, tags, meta, fields, time)` where `tags` and `meta` are both of type `map[string]string`.

You can copy a CCMessage with `FromMessage(other CCMessage) CCMessage`. To parse InfluxDB line protocol data, use `FromBytes(data []byte) ([]CCMessage, error)` which decodes one or more messages from line protocol format.

Although the [cc-specifications](https://github.com/ClusterCockpit/cc-specifications/blob/master/interfaces/lineprotocol/README.md) defines that there is only a `value` field for the metric value, the CCMessage still can have multiple values similar to the InfluxDB line protocol.

## Design Decisions

### Meta vs Tags

CCMessage extends InfluxDB line protocol with a separate `meta` map in addition to `tags`. This separation serves important purposes:

- **Tags** are used for filtering and querying in time-series databases. They're indexed and optimized for search operations.
- **Meta** fields contain descriptive metadata that doesn't need to be indexed (e.g., `unit`, `scope`, `source`).

When converting to InfluxDB line protocol via `ToPoint()` or `ToLineProtocol()`, you control which meta fields should be promoted to tags using the `metaAsTags` parameter. This provides flexibility without forcing all metadata into the tag space.

**Example:**
```golang
msg, _ := ccMessage.NewMetric(
    "cpu_usage",
    map[string]string{"hostname": "node001", "type": "node"},  // Tags for querying
    map[string]string{"unit": "percent", "scope": "hwthread"}, // Meta for context
    75.5,
    time.Now(),
)

// Convert with unit as tag, scope remains in meta (not exported)
lp := msg.ToLineProtocol(map[string]bool{"unit": true})
```

### Tag Sorting

When serializing messages to InfluxDB line protocol (via `Bytes()` or `ToLineProtocol()`), tags are sorted alphabetically by key. This ensures:

1. **Deterministic output**: Same message always produces identical serialization
2. **Test reliability**: Output can be compared for equality in tests
3. **Cache efficiency**: Consistent ordering improves cache hit rates in downstream systems
4. **Line protocol compliance**: InfluxDB recommends sorted tags for optimal performance

The sorting happens only during serialization; tags are stored in an unsorted map internally for efficient access.

## Message Type Helpers

The `ccMessage` package provides convenient helper functions to create different types of messages:

### Metrics

Metrics are numerical measurements from the monitored system.

```golang
msg, err := ccMessage.NewMetric(
    "cpu_usage",                           // metric name
    map[string]string{"type": "node"},     // tags
    map[string]string{"unit": "percent"},  // meta
    75.5,                                  // value (numeric)
    time.Now(),                            // timestamp
)
```

### Events

Events represent significant occurrences in the system.

```golang
msg, err := ccMessage.NewEvent(
    "node_down",                          // event name
    map[string]string{"severity": "critical"}, // tags
    nil,                                  // meta
    "Node node001 is unreachable",        // event payload
    time.Now(),                           // timestamp
)
```

For job-related events:

```golang
// Job start event
startMsg, err := ccMessage.NewJobStartEvent(job)

// Job stop event
stopMsg, err := ccMessage.NewJobStopEvent(job)

// Check if message is a job event
if eventName, ok := msg.IsJobEvent(); ok {
    job, err := msg.GetJob()  // Deserialize job data
}
```

### Logs

Log messages transmit textual log data through the system.

```golang
msg, err := ccMessage.NewLog(
    "application_log",                       // log category
    map[string]string{"level": "error"},     // tags
    map[string]string{"source": "backend"}, // meta
    "Database connection failed: timeout",   // log message
    time.Now(),                              // timestamp
)
```

### Control Messages

Control messages request or set configuration values.

```golang
// GET control - request current value
getMsg, err := ccMessage.NewGetControl(
    "sampling_rate",  // parameter name
    nil,              // tags
    nil,              // meta
    time.Now(),       // timestamp
)

// PUT control - set new value
putMsg, err := ccMessage.NewPutControl(
    "sampling_rate",  // parameter name
    nil,              // tags
    nil,              // meta
    "10",             // new value
    time.Now(),       // timestamp
)
```

### Query Messages

Query messages contain database queries or search requests.

```golang
msg, err := ccMessage.NewQuery(
    "metrics_query",  // query name
    nil,              // tags
    nil,              // meta
    "SELECT * FROM metrics WHERE timestamp > NOW() - INTERVAL '1h'", // query string
    time.Now(),       // timestamp
)
```

## Type Detection

CCMessage provides methods to detect the message type:

```golang
// Check message type
switch msg.MessageType() {
case ccMessage.CCMSG_TYPE_METRIC:
    value := msg.GetMetricValue()
case ccMessage.CCMSG_TYPE_EVENT:
    event := msg.GetEventValue()
case ccMessage.CCMSG_TYPE_LOG:
    log := msg.GetLogValue()
case ccMessage.CCMSG_TYPE_CONTROL:
    value := msg.GetControlValue()
    method := msg.GetControlMethod()  // "GET" or "PUT"
}

// Or use individual type checks
if msg.IsMetric() {
    value := msg.GetMetricValue()
}
if msg.IsEvent() {
    payload := msg.GetEventValue()
}
if msg.IsLog() {
    logText := msg.GetLogValue()
}
if msg.IsControl() {
    value := msg.GetControlValue()
    method := msg.GetControlMethod()
}
if msg.IsQuery() {
    query := msg.GetQueryValue()
}
```

## Common Usage Patterns

### Working with Metrics

```golang
// Create a metric with validation
msg, err := ccMessage.NewMetric(
    "memory_used",
    map[string]string{
        "hostname": "node001",
        "type":     "node",
    },
    map[string]string{
        "unit":  "bytes",
        "scope": "node",
    },
    int64(8589934592), // 8 GB
    time.Now(),
)
if err != nil {
    log.Fatalf("Failed to create metric: %v", err)
}

// Access metric value with type checking
if value, ok := msg.GetMetricValue(); ok {
    switch v := value.(type) {
    case int64:
        fmt.Printf("Integer metric: %d\n", v)
    case uint64:
        fmt.Printf("Unsigned metric: %d\n", v)
    case float64:
        fmt.Printf("Float metric: %.2f\n", v)
    }
}

// Convert to InfluxDB line protocol with unit as tag
lineProtocol := msg.ToLineProtocol(map[string]bool{"unit": true})
fmt.Println(lineProtocol)
// Output: memory_used,hostname=node001,type=node,unit=bytes value=8589934592 1234567890000000000
```

### Parsing Line Protocol

```golang
// Parse InfluxDB line protocol data
data := []byte(`cpu_usage,hostname=node001,type=node value=75.5 1234567890000000000
mem_used,hostname=node001,type=node value=8192 1234567890000000000`)

messages, err := ccMessage.FromBytes(data)
if err != nil {
    log.Fatalf("Failed to parse: %v", err)
}

for _, msg := range messages {
    fmt.Printf("Metric: %s = %v\n", msg.Name(), msg.GetMetricValue())
}
```

### Handling Events with JSON Payloads

```golang
// Create event with structured data
eventData := map[string]interface{}{
    "node":      "node001",
    "status":    "down",
    "timestamp": time.Now().Unix(),
    "reason":    "network timeout",
}
jsonPayload, _ := json.Marshal(eventData)

event, err := ccMessage.NewEvent(
    "node_failure",
    map[string]string{"severity": "critical", "cluster": "production"},
    nil,
    string(jsonPayload),
    time.Now(),
)

// Parse event payload
if payload, ok := event.GetEventValue(); ok {
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(payload), &data); err == nil {
        fmt.Printf("Node %s is %s\n", data["node"], data["status"])
    }
}
```

### Message Transformation

```golang
// Clone and modify a message
original, _ := ccMessage.NewMetric("cpu_usage", nil, nil, 50.0, time.Now())

// Create independent copy
modified := ccMessage.FromMessage(original)
modified.AddTag("datacenter", "dc1")
modified.AddMeta("aggregated", "true")

// Original remains unchanged
fmt.Printf("Original tags: %v\n", original.Tags())   // map[]
fmt.Printf("Modified tags: %v\n", modified.Tags())   // map[datacenter:dc1]
```

### Working with Control Messages

```golang
// Request current sampling rate
getRequest, _ := ccMessage.NewGetControl(
    "sampling_rate",
    map[string]string{"component": "collector"},
    nil,
    time.Now(),
)

// Check control method
if method, ok := getRequest.GetControlMethod(); ok {
    fmt.Printf("Control method: %s\n", method) // "GET"
}

// Update sampling rate
putRequest, _ := ccMessage.NewPutControl(
    "sampling_rate",
    map[string]string{"component": "collector"},
    nil,
    "5",
    time.Now(),
)

if value, ok := putRequest.GetControlValue(); ok {
    fmt.Printf("New value: %s\n", value) // "5"
}
```

### Batch Processing

```golang
// Process multiple messages
metrics := []struct {
    name  string
    value float64
}{
    {"cpu_usage", 75.5},
    {"mem_usage", 82.3},
    {"disk_usage", 45.1},
}

var messages []ccMessage.CCMessage
for _, m := range metrics {
    msg, err := ccMessage.NewMetric(
        m.name,
        map[string]string{"hostname": "node001"},
        map[string]string{"unit": "percent"},
        m.value,
        time.Now(),
    )
    if err != nil {
        log.Printf("Skipping metric %s: %v", m.name, err)
        continue
    }
    messages = append(messages, msg)
}

// Convert all to line protocol
for _, msg := range messages {
    lp := msg.ToLineProtocol(map[string]bool{"unit": true})
    fmt.Println(lp)
}
```

### Type-Safe Message Handling

```golang
func processMessage(msg ccMessage.CCMessage) {
    switch msg.MessageType() {
    case ccMessage.CCMSG_TYPE_METRIC:
        if value, ok := msg.GetMetricValue(); ok {
            fmt.Printf("Processing metric %s: %v\n", msg.Name(), value)
            // Send to time-series database
        }
    
    case ccMessage.CCMSG_TYPE_EVENT:
        if event, ok := msg.GetEventValue(); ok {
            fmt.Printf("Processing event %s: %s\n", msg.Name(), event)
            // Send to event log
        }
    
    case ccMessage.CCMSG_TYPE_LOG:
        if logMsg, ok := msg.GetLogValue(); ok {
            fmt.Printf("Processing log %s: %s\n", msg.Name(), logMsg)
            // Send to logging system
        }
    
    case ccMessage.CCMSG_TYPE_CONTROL:
        if method, ok := msg.GetControlMethod(); ok {
            value, _ := msg.GetControlValue()
            fmt.Printf("Control %s %s = %s\n", method, msg.Name(), value)
            // Handle configuration change
        }
    
    default:
        fmt.Printf("Unknown message type: %s\n", msg.Name())
    }
}
```

## Error Handling and Validation

### Input Validation

All message creation functions perform validation and return errors for invalid inputs:

```golang
// Empty names are rejected
msg, err := ccMessage.NewMetric("", nil, nil, 123, time.Now())
// Error: message name cannot be empty

// Zero timestamps are rejected
msg, err := ccMessage.NewMetric("test", nil, nil, 123, time.Time{})
// Error: timestamp cannot be zero

// Empty keys are rejected
msg, err := ccMessage.NewMetric("test",
    map[string]string{"": "value"}, // empty tag key
    nil, 123, time.Now())
// Error: tag keys cannot be empty

// NaN and Inf values are rejected
msg, err := ccMessage.NewMetric("test", nil, nil, math.NaN(), time.Now())
// Error: field 'value' has invalid float value (NaN or Inf)

// At least one field is required
msg, err := ccMessage.NewMessage("test", nil, nil,
    map[string]any{}, // no fields
    time.Now())
// Error: at least one field is required
```

### Type Conversion and Handling

```golang
// Automatic type conversion
msg, _ := ccMessage.NewMetric("test", nil, nil, int32(100), time.Now())
value, _ := msg.GetMetricValue()
// value is int64(100), not int32

// Unsupported types become nil and are skipped
type customType struct{ value int }
msg, err := ccMessage.NewMessage("test", nil, nil,
    map[string]any{
        "valid": 123,
        "invalid": customType{42}, // unsupported type
    },
    time.Now())
// err == nil, but "invalid" field is not present in message

// Checking for nil pointer values
var ptr *int64 = nil
msg, _ := ccMessage.NewMessage("test", nil, nil,
    map[string]any{"value": ptr},
    time.Now())
// "value" field will not be present (nil pointers are skipped)
```

### Safe Type Assertions

```golang
// Always use the ok pattern
if value, ok := msg.GetMetricValue(); ok {
    // Safe to use value
    fmt.Printf("Metric value: %v\n", value)
} else {
    // Not a metric or no value field
    fmt.Println("Not a metric message")
}

// Type-specific value retrieval already checks type
if logMsg, ok := msg.GetLogValue(); ok {
    // Guaranteed to be a string
    fmt.Println(logMsg)
}

// Don't panic on type assertions
value, ok := msg.GetMetricValue()
if !ok {
    return errors.New("expected metric message")
}
// Now safe to use value
```

### Concurrent Access Patterns

```golang
// WRONG: Concurrent modification without synchronization
msg, _ := ccMessage.NewMetric("test", nil, nil, 0.0, time.Now())
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(val int) {
        defer wg.Done()
        msg.AddTag(fmt.Sprintf("tag%d", val), "value") // RACE CONDITION!
    }(i)
}
wg.Wait()

// CORRECT: Use mutex for synchronization
var mu sync.Mutex
msg, _ := ccMessage.NewMetric("test", nil, nil, 0.0, time.Now())
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(val int) {
        defer wg.Done()
        mu.Lock()
        msg.AddTag(fmt.Sprintf("tag%d", val), "value")
        mu.Unlock()
    }(i)
}
wg.Wait()

// BETTER: Create separate messages per goroutine
original, _ := ccMessage.NewMetric("test", nil, nil, 0.0, time.Now())
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(val int) {
        defer wg.Done()
        // Each goroutine gets its own copy
        msg := ccMessage.FromMessage(original)
        msg.AddTag(fmt.Sprintf("tag%d", val), "value")
        // Process msg independently
    }(i)
}
wg.Wait()
```

### Serialization Error Handling

```golang
// Handle serialization errors
msg, _ := ccMessage.NewMetric("test", nil, nil, 123.45, time.Now())

// Line protocol conversion
lp := msg.ToLineProtocol(map[string]bool{})
// Line protocol conversion doesn't return errors (uses panic recovery)

// Bytes conversion can fail
bytes, err := msg.(*ccmessage.ccMessage).Bytes()
if err != nil {
    log.Printf("Serialization failed: %v", err)
    // Error might indicate unsupported field type or encoding issue
}

// JSON conversion can fail
json, err := msg.ToJSON(map[string]bool{})
if err != nil {
    log.Printf("JSON conversion failed: %v", err)
}

// Parsing can fail with detailed errors
data := []byte("invalid line protocol !!!")
messages, err := ccMessage.FromBytes(data)
if err != nil {
    log.Printf("Failed to parse: %v", err)
    // Error will indicate what went wrong (invalid measurement, tags, etc.)
}
```

## Best Practices

1. **Use appropriate message types**: Choose the correct message type for your data. Use metrics for numerical measurements, events for significant occurrences, logs for textual output, and control messages for configuration.

2. **Leverage tags for categorization**: Use tags to categorize messages for efficient filtering and querying. Common tags include `type`, `hostname`, `cluster`, `severity`.

3. **Store metadata in meta fields**: Use meta fields for information that describes the data but shouldn't be used for filtering, such as `unit`, `scope`, `source`.

4. **Handle timestamps consistently**: Always use appropriate timestamps for your messages. For job events, use the job's actual start time.

5. **Validate before use**: Check error returns from message creation functions and type assertion operations.

6. **Deep copy when needed**: Use `FromMessage()` to create independent copies of messages when you need to modify them without affecting the original.

7. **Type checking**: Use the type detection methods (`IsMetric()`, `IsEvent()`, etc.) before accessing type-specific values to avoid runtime errors.

8. **Thread safety**: CCMessage instances are NOT thread-safe. If you need to access a message from multiple goroutines, either use external synchronization (mutexes) or create separate copies with `FromMessage()` for each goroutine.

