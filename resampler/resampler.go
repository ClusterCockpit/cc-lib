// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package resampler

import (
	"errors"
	"fmt"
	"math"

	"github.com/ClusterCockpit/cc-lib/schema"
)

func calculateTriangleArea(paX, paY, pbX, pbY, pcX, pcY schema.Float) float64 {
	area := ((paX-pcX)*(pbY-paY) - (paX-pbX)*(pcY-paY)) * 0.5
	return math.Abs(float64(area))
}

func calculateAverageDataPoint(points []schema.Float, xStart int64) (avgX schema.Float, avgY schema.Float) {
	flag := 0
	for _, point := range points {
		avgX += schema.Float(xStart)
		avgY += point
		xStart++
		if math.IsNaN(float64(point)) {
			flag = 1
		}
	}

	l := schema.Float(len(points))

	avgX /= l
	avgY /= l

	if flag == 1 {
		return avgX, schema.NaN
	} else {
		return avgX, avgY
	}
}

func SimpleResampler(data []schema.Float, old_frequency int64, new_frequency int64) ([]schema.Float, int64, error) {
	if old_frequency == 0 || new_frequency == 0 || new_frequency <= old_frequency {
		return data, old_frequency, nil
	}

	if new_frequency%old_frequency != 0 {
		return nil, 0, errors.New("new sampling frequency should be multiple of the old frequency")
	}

	var step int = int(new_frequency / old_frequency)
	new_data_length := len(data) / step

	if new_data_length == 0 || len(data) < 100 || new_data_length >= len(data) {
		return data, old_frequency, nil
	}

	new_data := make([]schema.Float, new_data_length)

	for i := 0; i < new_data_length; i++ {
		new_data[i] = data[i*step]
	}

	return new_data, new_frequency, nil
}

// Inspired by one of the algorithms from https://skemman.is/bitstream/1946/15343/3/SS_MSthesis.pdf
// Adapted from https://github.com/haoel/downsampling/blob/master/core/lttb.go
func LargestTriangleThreeBucket(data []schema.Float, old_frequency int, new_frequency int) ([]schema.Float, int, error) {
	if old_frequency == 0 || new_frequency == 0 || new_frequency <= old_frequency {
		return data, old_frequency, nil
	}

	if new_frequency%old_frequency != 0 {
		return nil, 0, errors.New(fmt.Sprintf("new sampling frequency : %d should be multiple of the old frequency : %d", new_frequency, old_frequency))
	}

	var step int = int(new_frequency / old_frequency)
	new_data_length := len(data) / step

	if new_data_length == 0 || len(data) < 100 || new_data_length >= len(data) {
		return data, old_frequency, nil
	}

	new_data := make([]schema.Float, 0, new_data_length)

	// Bucket size. Leave room for start and end data points
	bucketSize := float64(len(data)-2) / float64(new_data_length-2)

	new_data = append(new_data, data[0]) // Always add the first point

	// We have 3 pointers represent for
	// > bucketLow - the current bucket's beginning location
	// > bucketMiddle - the current bucket's ending location,
	//                  also the beginning location of next bucket
	// > bucketHight - the next bucket's ending location.
	bucketLow := 1
	bucketMiddle := int(math.Floor(bucketSize)) + 1

	var prevMaxAreaPoint int

	for i := 0; i < new_data_length-2; i++ {

		bucketHigh := int(math.Floor(float64(i+2)*bucketSize)) + 1
		if bucketHigh >= len(data)-1 {
			bucketHigh = len(data) - 2
		}

		// Calculate point average for next bucket (containing c)
		avgPointX, avgPointY := calculateAverageDataPoint(data[bucketMiddle:bucketHigh+1], int64(bucketMiddle))

		// Get the range for current bucket
		currBucketStart := bucketLow
		currBucketEnd := bucketMiddle

		// Point a
		pointX := prevMaxAreaPoint
		pointY := data[prevMaxAreaPoint]

		maxArea := -1.0

		var maxAreaPoint int
		flag_ := 0
		for ; currBucketStart < currBucketEnd; currBucketStart++ {

			area := calculateTriangleArea(schema.Float(pointX), pointY, avgPointX, avgPointY, schema.Float(currBucketStart), data[currBucketStart])
			if area > maxArea {
				maxArea = area
				maxAreaPoint = currBucketStart
			}
			if math.IsNaN(float64(avgPointY)) {
				flag_ = 1
			}
		}

		if flag_ == 1 {
			new_data = append(new_data, schema.NaN) // Pick this point from the bucket
		} else {
			new_data = append(new_data, data[maxAreaPoint]) // Pick this point from the bucket
		}
		prevMaxAreaPoint = maxAreaPoint // This MaxArea point is the next's prevMAxAreaPoint

		// move to the next window
		bucketLow = bucketMiddle
		bucketMiddle = bucketHigh
	}

	new_data = append(new_data, data[len(data)-1]) // Always add last

	return new_data, new_frequency, nil
}
