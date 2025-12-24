# Resampler Package

The `resampler` package provides efficient time-series data downsampling algorithms for reducing the number of data points while preserving important characteristics of the data.

## Overview

When working with time-series data, it's often necessary to reduce the number of data points for visualization, storage, or transmission purposes. This package implements two downsampling strategies:

1. **SimpleResampler**: Fast, straightforward downsampling by selecting every nth point
2. **LargestTriangleThreeBucket (LTTB)**: Perceptually-aware downsampling that preserves visual characteristics

## Algorithms

### SimpleResampler

The `SimpleResampler` function performs simple downsampling by selecting every nth point from the input data.

**Characteristics:**
- **Speed**: Fastest algorithm, O(n) time complexity
- **Quality**: May miss important features like peaks and valleys
- **Use case**: When speed is critical and data is relatively uniform

**Example:**
```go
import "github.com/ClusterCockpit/cc-lib/v2/resampler"
import "github.com/ClusterCockpit/cc-lib/v2/schema"

data := []schema.Float{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
downsampled, newFreq, err := resampler.SimpleResampler(data, 1, 2)
if err != nil {
    log.Fatal(err)
}
// downsampled contains every 2nd point: [1.0, 3.0, 5.0, 7.0]
```

### LargestTriangleThreeBucket (LTTB)

The `LargestTriangleThreeBucket` function implements a sophisticated downsampling algorithm that preserves the visual characteristics of time-series data by selecting points that form the largest triangles with their neighbors.

**Characteristics:**
- **Speed**: Still efficient, O(n) time complexity
- **Quality**: Excellent preservation of peaks, valleys, and trends
- **Use case**: When visual fidelity is important (charts, graphs, monitoring dashboards)

**How it works:**
1. The data is divided into buckets (first and last points are always kept)
2. For each bucket, the algorithm selects the point that forms the largest triangle with:
   - The previously selected point
   - The average of the next bucket
3. This maximizes visual area and preserves important features

**Example:**
```go
import "github.com/ClusterCockpit/cc-lib/v2/resampler"
import "github.com/ClusterCockpit/cc-lib/v2/schema"

// Generate some sample data with peaks
data := make([]schema.Float, 1000)
for i := range data {
    data[i] = schema.Float(math.Sin(float64(i) * 0.1))
}

// Downsample from 1000 points to 100 points
downsampled, newFreq, err := resampler.LargestTriangleThreeBucket(data, 1, 10)
if err != nil {
    log.Fatal(err)
}
// downsampled contains 100 points that preserve the sine wave's visual characteristics
```

## API Reference

### SimpleResampler

```go
func SimpleResampler(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error)
```

**Parameters:**
- `data`: Input time-series data points
- `oldFrequency`: Original sampling frequency (points per time unit)
- `newFrequency`: Target sampling frequency (must be a multiple of oldFrequency)

**Returns:**
- Downsampled data slice
- Actual frequency used (may be oldFrequency if downsampling wasn't performed)
- Error if newFrequency is not a multiple of oldFrequency

### LargestTriangleThreeBucket

```go
func LargestTriangleThreeBucket(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error)
```

**Parameters:**
- `data`: Input time-series data points
- `oldFrequency`: Original sampling frequency (points per time unit)
- `newFrequency`: Target sampling frequency (must be a multiple of oldFrequency)

**Returns:**
- Downsampled data slice
- Actual frequency used (may be oldFrequency if downsampling wasn't performed)
- Error if newFrequency is not a multiple of oldFrequency

## Behavior Notes

Both functions will return the original data unchanged if:
- Either frequency is 0
- `newFrequency <= oldFrequency` (no downsampling needed)
- The resulting data would have fewer than 1 point
- The original data has fewer than 100 points
- The downsampled data would have the same or more points than the original

## NaN Handling

Both algorithms properly handle `NaN` (Not a Number) values:
- `SimpleResampler`: Preserves NaN values if they fall on selected points
- `LargestTriangleThreeBucket`: Considers NaN values in area calculations and preserves them appropriately

## Performance Comparison

| Algorithm | Time Complexity | Space Complexity | Visual Quality | Speed |
|-----------|----------------|------------------|----------------|-------|
| SimpleResampler | O(n) | O(m) | Good | Fastest |
| LTTB | O(n) | O(m) | Excellent | Fast |

Where:
- n = number of input points
- m = number of output points

Benchmark results (10,000 input points → 1,000 output points):
```
BenchmarkSimpleResampler-8                  50000    ~25000 ns/op
BenchmarkLargestTriangleThreeBucket-8       10000    ~120000 ns/op
```

LTTB is approximately 4-5x slower than SimpleResampler but still very fast for most use cases.

## When to Use Which Algorithm

**Use SimpleResampler when:**
- Speed is the primary concern
- Data is relatively uniform without important features
- You need the absolute fastest downsampling
- Visual quality is not critical

**Use LargestTriangleThreeBucket when:**
- Displaying data in charts or graphs
- Visual fidelity is important
- Data contains peaks, valleys, or trends that must be preserved
- You're building monitoring dashboards or visualization tools
- The slight performance overhead is acceptable

## References

- **LTTB Algorithm Paper**: [Downsampling Time Series for Visual Representation](https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf) by Sveinn Steinarsson
- **Original Implementation**: [haoel/downsampling](https://github.com/haoel/downsampling)

## License

Copyright (C) NHR@FAU, University Erlangen-Nuremberg.  
Licensed under the MIT License. See LICENSE file for details.
