// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"math"
	"strconv"
)

// Float is a custom float64 type that properly handles NaN values in JSON.
// Go's JSON encoder for floats does not support NaN (https://github.com/golang/go/issues/3480).
// This program uses NaN as a signal for missing data.
// For the HTTP JSON API to be able to handle NaN values,
// Float implements encoding/json.Marshaler and encoding/json.Unmarshaler,
// converting NaN to/from JSON null.
type Float float64

var (
	// NaN is a Float constant representing Not-a-Number, used to signal missing data.
	NaN         Float  = Float(math.NaN())
	nullAsBytes []byte = []byte("null")
)

// IsNaN returns true if this Float value represents NaN (Not-a-Number).
func (f Float) IsNaN() bool {
	return math.IsNaN(float64(f))
}

// MarshalJSON implements json.Marshaler.
// It converts NaN values to JSON null, and normal float values to JSON numbers with 3 decimal places.
func (f Float) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(f)) {
		return nullAsBytes, nil
	}

	return strconv.AppendFloat(make([]byte, 0, 10), float64(f), 'f', 3, 64), nil
}

// Double converts a Float to a native float64 value.
func (f Float) Double() float64 {
	return float64(f)
}

// UnmarshalJSON implements json.Unmarshaler.
// It converts JSON null to NaN, and normal JSON numbers to Float values.
func (f *Float) UnmarshalJSON(input []byte) error {
	if string(input) == "null" {
		*f = NaN
		return nil
	}

	val, err := strconv.ParseFloat(string(input), 64)
	if err != nil {
		return err
	}
	*f = Float(val)
	return nil
}

// FloatArray is a slice of Float values with optimized JSON marshaling.
// It can be marshaled to JSON with fewer allocations than []Float.
type FloatArray []Float

// ConvertToFloat converts a float64 to a Float.
// It treats -1.0 as a special value and converts it to NaN.
func ConvertToFloat(input float64) Float {
	if input == -1.0 {
		return NaN
	} else {
		return Float(input)
	}
}

// MarshalJSON implements json.Marshaler for FloatArray.
// It efficiently marshals a slice of Float values, converting NaN to null.
func (fa FloatArray) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 2+len(fa)*8)
	buf = append(buf, '[')
	for i := range fa {
		if i != 0 {
			buf = append(buf, ',')
		}

		if fa[i].IsNaN() {
			buf = append(buf, `null`...)
		} else {
			buf = strconv.AppendFloat(buf, float64(fa[i]), 'f', 3, 64)
		}
	}
	buf = append(buf, ']')
	return buf, nil
}
