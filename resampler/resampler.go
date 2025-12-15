// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package resampler provides time-series data downsampling algorithms.
//
// This package implements two downsampling strategies for reducing the number
// of data points in time-series data while preserving important characteristics:
//
//   - SimpleResampler: A fast, straightforward algorithm that selects every nth point
//   - LargestTriangleThreeBucket (LTTB): A perceptually-aware algorithm that preserves
//     visual characteristics by selecting points that maximize the area of triangles
//     formed with neighboring points
//
// Both algorithms are designed to work with schema.Float data and handle NaN values
// appropriately. They require that the new sampling frequency is a multiple of the
// old frequency.
//
// References:
//   - LTTB Algorithm: https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf
//   - Implementation adapted from: https://github.com/haoel/downsampling
package resampler

import (
	"fmt"
	"math"

	"github.com/ClusterCockpit/cc-lib/schema"
)

// Default number of points required to trigger resampling.
// Otherwise, time series of original timestep will be returned without resampling
var MinimumRequiredPoints int = 1000

// calculateTriangleArea computes the area of a triangle defined by three points.
//
// The area is calculated using the cross product formula:
// Area = 0.5 * |((paX - pcX) * (pbY - paY)) - ((paX - pbX) * (pcY - paY))|
//
// This is used by the LTTB algorithm to determine which points preserve the
// most visual information when downsampling.
func calculateTriangleArea(paX, paY, pbX, pbY, pcX, pcY schema.Float) float64 {
	area := ((paX-pcX)*(pbY-paY) - (paX-pbX)*(pcY-paY)) * 0.5
	return math.Abs(float64(area))
}

// calculateAverageDataPoint computes the average point from a slice of data points.
//
// Parameters:
//   - points: slice of Y values to average
//   - xStart: starting X coordinate for the points
//
// Returns:
//   - avgX: average X coordinate
//   - avgY: average Y value, or NaN if any input point is NaN
//
// This function is used by LTTB to find the centroid of points in a bucket,
// which is then used to calculate triangle areas for point selection.
func calculateAverageDataPoint(points []schema.Float, xStart int64) (avgX schema.Float, avgY schema.Float) {
	hasNaN := false
	for _, point := range points {
		avgX += schema.Float(xStart)
		avgY += point
		xStart++
		if math.IsNaN(float64(point)) {
			hasNaN = true
		}
	}

	l := schema.Float(len(points))
	avgX /= l
	avgY /= l

	if hasNaN {
		return avgX, schema.NaN
	}
	return avgX, avgY
}

// SimpleResampler performs simple downsampling by selecting every nth point.
//
// This is the fastest downsampling method but may miss important features in the data.
// It works by calculating a step size (newFrequency / oldFrequency) and selecting
// every step-th point from the original data.
//
// Parameters:
//   - data: input time-series data points
//   - oldFrequency: original sampling frequency (points per time unit)
//   - newFrequency: target sampling frequency (must be a multiple of oldFrequency)
//
// Returns:
//   - Downsampled data slice
//   - Actual frequency used (may be oldFrequency if downsampling wasn't performed)
//   - Error if newFrequency is not a multiple of oldFrequency
//
// The function returns the original data unchanged if:
//   - Either frequency is 0
//   - newFrequency <= oldFrequency (no downsampling needed)
//   - The resulting data would have fewer than 1 point
//   - The original data has fewer than 100 points
//   - The downsampled data would have the same or more points than the original
//
// Example:
//
//	data := []schema.Float{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
//	downsampled, freq, err := SimpleResampler(data, 1, 2)
//	// Returns: [1.0, 3.0, 5.0], 2, nil
func SimpleResampler(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error) {
	// checks if the frequencies are valid or not.
	newDataLength, step := validateFrequency(len(data), oldFrequency, newFrequency)
	if newDataLength == -1 {
		return data, oldFrequency, nil
	}

	newData := make([]schema.Float, newDataLength)
	for i := 0; i < newDataLength; i++ {
		newData[i] = data[i*step]
	}

	return newData, newFrequency, nil
}

func validateFrequency(lenData int, oldFrequency, newFrequency int64) (int, int) {
	// Validate inputs and check if downsampling is needed
	if oldFrequency == 0 || newFrequency == 0 || newFrequency <= oldFrequency {
		return -1, 0
	}

	// Ensure new frequency is a multiple of old frequency
	if newFrequency%oldFrequency != 0 {
		fmt.Printf("new sampling frequency (%d) must be a multiple of old frequency (%d)", newFrequency, oldFrequency)
		return -1, 0
	}

	step := int(newFrequency / oldFrequency)
	newDataLength := lenData / step

	// Don't downsample if result would be trivial or counterproductive
	if (newDataLength == 0) || (lenData < MinimumRequiredPoints) || (newDataLength >= lenData) {
		return -1, 0
	}

	return newDataLength, step
}

// LargestTriangleThreeBucket (LTTB) performs perceptually-aware downsampling.
//
// LTTB is a downsampling algorithm that preserves the visual characteristics of
// time-series data by selecting points that form the largest triangles with their
// neighbors. This ensures that important peaks, valleys, and trends are retained
// even when significantly reducing the number of points.
//
// Algorithm Overview:
//  1. The data is divided into buckets (except first and last points which are always kept)
//  2. For each bucket, the algorithm selects the point that forms the largest triangle
//     with the previous selected point and the average of the next bucket
//  3. This maximizes the visual area and preserves important features
//
// Time Complexity: O(n) where n is the number of input points
// Space Complexity: O(m) where m is the number of output points
//
// Parameters:
//   - data: input time-series data points
//   - oldFrequency: original sampling frequency (points per time unit)
//   - newFrequency: target sampling frequency (must be a multiple of oldFrequency)
//
// Returns:
//   - Downsampled data slice
//   - Actual frequency used (may be oldFrequency if downsampling wasn't performed)
//   - Error if newFrequency is not a multiple of oldFrequency
//
// The function returns the original data unchanged if:
//   - Either frequency is 0
//   - newFrequency <= oldFrequency (no downsampling needed)
//   - The resulting data would have fewer than 1 point
//   - The original data has fewer than 100 points
//   - The downsampled data would have the same or more points than the original
//
// NaN Handling:
// The algorithm properly handles NaN values and preserves them in the output when
// they represent the maximum area point in a bucket.
//
// Example:
//
//	data := []schema.Float{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0}
//	downsampled, freq, err := LargestTriangleThreeBucket(data, 1, 2)
//	// Returns downsampled data preserving visual characteristics
//
// References:
//   - Original paper: https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf
//   - Adapted from: https://github.com/haoel/downsampling/blob/master/core/lttb.go
func LargestTriangleThreeBucket(data []schema.Float, oldFrequency int64, newFrequency int64) ([]schema.Float, int64, error) {
	// checks if the frequencies are valid or not.
	newDataLength, _ := validateFrequency(len(data), oldFrequency, newFrequency)
	if newDataLength == -1 {
		return data, oldFrequency, nil
	}

	newData := make([]schema.Float, 0, newDataLength)

	// Bucket size. Leave room for start and end data points
	bucketSize := float64(len(data)-2) / float64(newDataLength-2)

	// Always add the first point
	newData = append(newData, data[0])

	// Bucket pointers:
	// - bucketLow: current bucket's start
	// - bucketMiddle: current bucket's end (also next bucket's start)
	// - bucketHigh: next bucket's end
	bucketLow := 1
	bucketMiddle := int(math.Floor(bucketSize)) + 1

	var prevMaxAreaPoint int

	// Process each bucket (excluding first and last points)
	for i := 0; i < newDataLength-2; i++ {
		bucketHigh := int(math.Floor(float64(i+2)*bucketSize)) + 1
		if bucketHigh >= len(data)-1 {
			bucketHigh = len(data) - 2
		}

		// Calculate average point for next bucket (point c in triangle)
		avgPointX, avgPointY := calculateAverageDataPoint(data[bucketMiddle:bucketHigh+1], int64(bucketMiddle))

		// Get the range for current bucket
		currBucketStart := bucketLow
		currBucketEnd := bucketMiddle

		// Point a (previously selected point)
		pointX := prevMaxAreaPoint
		pointY := data[prevMaxAreaPoint]

		maxArea := -1.0
		var maxAreaPoint int
		flag_ := 0

		// Find the point in current bucket that forms the largest triangle
		for ; currBucketStart < currBucketEnd; currBucketStart++ {
			area := calculateTriangleArea(
				schema.Float(pointX), pointY,
				avgPointX, avgPointY,
				schema.Float(currBucketStart), data[currBucketStart],
			)

			if area > maxArea || math.IsNaN(area) {
				maxArea = area
				maxAreaPoint = currBucketStart
			}
			// if math.IsNaN(float64(avgPointY)) {
			// 	flag_ = 1
			// }
		}

		// Add the point with maximum area from this bucket
		if flag_ == 1 {
			newData = append(newData, schema.NaN) // Pick this point from the bucket
		} else {
			newData = append(newData, data[maxAreaPoint]) // Pick this point from the bucket
		}

		prevMaxAreaPoint = maxAreaPoint

		// Move to the next bucket
		bucketLow = bucketMiddle
		bucketMiddle = bucketHigh
	}

	// Always add the last point
	newData = append(newData, data[len(data)-1])

	return newData, newFrequency, nil
}

func SetMinimumRequiredPoints(setVal int) {
	MinimumRequiredPoints = setVal
}
