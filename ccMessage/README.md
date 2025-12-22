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
func FromInfluxMetric(other lp.Metric) CCMessage
```

The `CCMessage` interface provides the same functions as the `MutableMetric` like `{Add, Get, Remove, Has}{Tag, Field}` and additionally provides `{Add, Get, Remove, Has}Meta`.

The InfluxDB protocol creates a new metric with `influx.New(name, tags, fields, time)` while CCMessage uses `ccMessage.New(name, tags, meta, fields, time)` where `tags` and `meta` are both of type `map[string]string`.

You can copy a CCMessage with `FromFromMessage(other CCMessage) CCMessage`. If you get an `influx.Metric` from a function, like the line protocol parser, you can use `FromInfluxMetric(other influx.Metric) CCMessage` to get a CCMessage out of it (see `NatsReceiver` for an example).

Although the [cc-specifications](https://github.com/ClusterCockpit/cc-specifications/blob/master/interfaces/lineprotocol/README.md) defines that there is only a `value` field for the metric value, the CCMessage still can have multiple values similar to the InfluxDB line protocol.

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

## Best Practices

1. **Use appropriate message types**: Choose the correct message type for your data. Use metrics for numerical measurements, events for significant occurrences, logs for textual output, and control messages for configuration.

2. **Leverage tags for categorization**: Use tags to categorize messages for efficient filtering and querying. Common tags include `type`, `hostname`, `cluster`, `severity`.

3. **Store metadata in meta fields**: Use meta fields for information that describes the data but shouldn't be used for filtering, such as `unit`, `scope`, `source`.

4. **Handle timestamps consistently**: Always use appropriate timestamps for your messages. For job events, use the job's actual start time.

5. **Validate before use**: Check error returns from message creation functions and type assertion operations.

6. **Deep copy when needed**: Use `FromMessage()` to create independent copies of messages when you need to modify them without affecting the original.

7. **Type checking**: Use the type detection methods (`IsMetric()`, `IsEvent()`, etc.) before accessing type-specific values to avoid runtime errors.

8. **Thread safety**: CCMessage instances are NOT thread-safe. If you need to access a message from multiple goroutines, either use external synchronization (mutexes) or create separate copies with `FromMessage()` for each goroutine.

