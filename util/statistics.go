// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"cmp"
	"fmt"
	"math"
	"sort"
)

// Min returns the minimum of two values of any ordered type.
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two values of any ordered type.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// sortedCopy creates a sorted copy of a float64 slice without modifying the original.
func sortedCopy(input []float64) []float64 {
	sorted := make([]float64, len(input))
	copy(sorted, input)
	sort.Float64s(sorted)
	return sorted
}

// Mean calculates the arithmetic mean (average) of a float64 slice.
// Returns NaN and an error if the input slice is empty.
func Mean(input []float64) (float64, error) {
	if len(input) == 0 {
		return math.NaN(), fmt.Errorf("input array is empty: %#v", input)
	}
	sum := 0.0
	for _, n := range input {
		sum += n
	}
	return sum / float64(len(input)), nil
}

// Median calculates the median value of a float64 slice.
// For even-length slices, it returns the mean of the two middle values.
// For odd-length slices, it returns the middle value.
// Returns NaN and an error if the input slice is empty.
func Median(input []float64) (median float64, err error) {
	c := sortedCopy(input)
	// Even numbers: add the two middle numbers, divide by two (use mean function)
	// Odd numbers: Use the middle number
	l := len(c)
	if l == 0 {
		return math.NaN(), fmt.Errorf("input array is empty: %#v", input)
	} else if l%2 == 0 {
		median, _ = Mean(c[l/2-1 : l/2+1])
	} else {
		median = c[l/2]
	}
	return median, nil
}
