# Resampler Package

The `resampler` package provides efficient time-series data downsampling
algorithms for reducing the number of data points while preserving important
characteristics of the data.

## Overview

When working with time-series data, it's often necessary to reduce the number of
data points for visualization, storage, or transmission purposes. This package
implements three downsampling strategies:

1. **SimpleResampler**: Fast, straightforward downsampling by selecting every nth point
2. **LargestTriangleThreeBucket (LTTB)**: Perceptually-aware downsampling that preserves visual characteristics
3. **AverageResampler**: RRDTool-style consolidation by averaging points within each bucket

## Algorithms

### SimpleResampler

The `SimpleResampler` function performs simple downsampling by selecting every
nth point from the input data.

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

The `LargestTriangleThreeBucket` function implements a sophisticated
downsampling algorithm that preserves the visual characteristics of time-series
data by selecting points that form the largest triangles with their neighbors.

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

### AverageResampler

The `AverageResampler` function performs RRDTool-style average consolidation by
dividing data into fixed-size buckets and computing the arithmetic mean of all
valid (non-NaN) points in each bucket.

**Characteristics:**

- **Speed**: Fast, O(n) time complexity
- **Quality**: Scientifically accurate averages; smooths noise but may lose sharp peaks
- **Use case**: When statistical accuracy over time intervals matters more than visual shape preservation

**NaN handling:** NaN values are skipped when computing the bucket average. If
all values in a bucket are NaN, the output for that bucket is NaN.

**Example:**

```go
import "github.com/ClusterCockpit/cc-lib/v2/resampler"
import "github.com/ClusterCockpit/cc-lib/v2/schema"

data := []schema.Float{1.0, 3.0, 2.0, 4.0, 5.0, 3.0}
downsampled, newFreq, err := resampler.AverageResampler(data, 1, 2)
if err != nil {
    log.Fatal(err)
}
// downsampled contains bucket averages: [2.0, 3.0, 4.0]
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

### AverageResampler

```go
func AverageResampler(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error)
```

**Parameters:**

- `data`: Input time-series data points
- `oldFrequency`: Original sampling frequency
- `newFrequency`: Target sampling frequency (must be a multiple of oldFrequency)

**Returns:**

- Averaged data slice
- Actual frequency used
- Error if newFrequency is not a multiple of oldFrequency

### GetResampler

```go
func GetResampler(name string) (ResamplerFunc, error)
```

Returns the resampler function for the given name. Valid names are `"lttb"` (default), `"average"`, and `"simple"`. An empty string returns LTTB.

**Example:**

```go
fn, err := resampler.GetResampler("average")
if err != nil {
    log.Fatal(err)
}
downsampled, freq, err := fn(data, oldFreq, newFreq)
```

### ResamplerFunc

```go
type ResamplerFunc func(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error)
```

The function signature shared by all resampler algorithms. Use this type when storing or passing resampler functions.

### MinimumRequiredPoints / SetMinimumRequiredPoints

```go
var MinimumRequiredPoints int = 1000

func SetMinimumRequiredPoints(setVal int)
```

`MinimumRequiredPoints` is the minimum number of input points required to trigger resampling. If the input has fewer points, all three algorithms return the original data unchanged. The default is **1000**. Use `SetMinimumRequiredPoints` to override the threshold.

## Behavior Notes

All three functions return the original data unchanged if:

- Either frequency is 0
- `newFrequency <= oldFrequency` (no downsampling needed)
- The resulting data would have fewer than 1 point
- The original data has fewer than `MinimumRequiredPoints` points (default: 1000)
- The downsampled data would have the same or more points than the original

## NaN Handling

- **SimpleResampler**: Preserves NaN values if they fall on selected points
- **LargestTriangleThreeBucket**: NaN values propagate through the triangle area calculation; a bucket containing NaN may have its NaN point selected as the maximum-area point
- **AverageResampler**: NaN values are skipped when computing the bucket average; buckets where all values are NaN produce a NaN output

## Performance Comparison

| Algorithm        | Time Complexity | Space Complexity | Visual Quality | Use Case                     |
| ---------------- | --------------- | ---------------- | -------------- | ---------------------------- |
| SimpleResampler  | O(n)            | O(m)             | Basic          | Speed-critical, uniform data |
| LTTB             | O(n)            | O(m)             | Excellent      | Charts, dashboards           |
| AverageResampler | O(n)            | O(m)             | Smooth         | Statistical accuracy         |

Where:

- n = number of input points
- m = number of output points

## When to Use Which Algorithm

**Use SimpleResampler when:**

- Speed is the primary concern
- Data is relatively uniform without important features
- Visual quality is not critical

**Use LargestTriangleThreeBucket when:**

- Displaying data in charts or graphs
- Visual fidelity is important
- Data contains peaks, valleys, or trends that must be preserved
- You're building monitoring dashboards or visualization tools

**Use AverageResampler when:**

- Statistical accuracy over time intervals is required
- You want RRDTool-compatible consolidation behavior
- Data has noise you want smoothed out in the downsampled result
- Peaks don't need to be preserved exactly

## References

- **LTTB Algorithm Paper**: [Downsampling Time Series for Visual Representation](https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf) by Sveinn Steinarsson
- **Original Implementation**: [haoel/downsampling](https://github.com/haoel/downsampling)

## License

Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
Licensed under the MIT License. See LICENSE file for details.
