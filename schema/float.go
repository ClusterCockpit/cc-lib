// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

import (
	"errors"
	"io"
	"math"
	"strconv"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

// A custom float type is used so that (Un)MarshalJSON and
// (Un)MarshalGQL can be overloaded and NaN/null can be used.
// The default behaviour of putting every nullable value behind
// a pointer has a bigger overhead.
type Float float64

// Same as `[]Float`, but can be marshaled to JSON with less allocations.
type FloatArray []Float

var (
	NaN         Float  = Float(math.NaN())
	nullAsBytes []byte = []byte("null")
)

func (f Float) IsNaN() bool {
	return math.IsNaN(float64(f))
}

func (f Float) Double() float64 {
	return float64(f)
}

func ConvertToFloat(input float64) Float {
	if input == -1.0 {
		return NaN
	} else {
		return Float(input)
	}
}

// NaN will be serialized to `null`.
func (f Float) MarshalJSON() ([]byte, error) {
	if f.IsNaN() {
		return nullAsBytes, nil
	}

	return strconv.AppendFloat(make([]byte, 0, 10), float64(f), 'f', 2, 64), nil
}

// `null` will be unserialized to NaN.
func (f *Float) UnmarshalJSON(input []byte) error {
	s := string(input)
	if s == "null" {
		*f = NaN
		return nil
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		cclog.Warn("Error while parsing custom float")
		return err
	}
	*f = Float(val)
	return nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface.
func (f *Float) UnmarshalGQL(v any) error {
	f64, ok := v.(float64)
	if !ok {
		return errors.New("invalid Float scalar")
	}

	*f = Float(f64)
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface.
// NaN will be serialized to `null`.
func (f Float) MarshalGQL(w io.Writer) {
	if f.IsNaN() {
		w.Write(nullAsBytes)
	} else {
		w.Write(strconv.AppendFloat(make([]byte, 0, 10), float64(f), 'f', 2, 64))
	}
}

// Only used via REST-API, not via GraphQL.
// This uses a lot less allocations per series,
// but it turns out that the performance increase
// from using this is not that big.
func (s *Series) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 512+len(s.Data)*8)
	buf = append(buf, `{"hostname":"`...)
	buf = append(buf, s.Hostname...)
	buf = append(buf, '"')
	if s.Id != nil {
		buf = append(buf, `,"id":"`...)
		buf = append(buf, *s.Id...)
		buf = append(buf, '"')
	}
	buf = append(buf, `,"statistics":{"min":`...)
	buf = strconv.AppendFloat(buf, s.Statistics.Min, 'f', 2, 64)
	buf = append(buf, `,"avg":`...)
	buf = strconv.AppendFloat(buf, s.Statistics.Avg, 'f', 2, 64)
	buf = append(buf, `,"max":`...)
	buf = strconv.AppendFloat(buf, s.Statistics.Max, 'f', 2, 64)
	buf = append(buf, '}')
	buf = append(buf, `,"data":[`...)
	for i := range s.Data {
		if i != 0 {
			buf = append(buf, ',')
		}

		if s.Data[i].IsNaN() {
			buf = append(buf, `null`...)
		} else {
			buf = strconv.AppendFloat(buf, float64(s.Data[i]), 'f', 2, 32)
		}
	}
	buf = append(buf, ']', '}')
	return buf, nil
}

func ConvertFloatToFloat64(s []Float) []float64 {
	fp := make([]float64, len(s))

	for i, val := range s {
		fp[i] = float64(val)
	}

	return fp
}

func GetFloat64ToFloat(s []float64) []Float {
	fp := make([]Float, len(s))

	for i, val := range s {
		fp[i] = Float(val)
	}

	return fp
}
