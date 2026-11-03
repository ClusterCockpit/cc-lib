// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccunits

import (
	"math"
	"reflect"
	"testing"
)

func TestInvalidUnit_Sentinel(t *testing.T) {
	if INVALID_UNIT.Valid() || INVALID_UNIT.GetPrefix() != InvalidPrefix || INVALID_UNIT.GetMeasure() != InvalidMeasure || INVALID_UNIT.GetUnitDenominator() != InvalidMeasure {
		t.Errorf("INVALID_UNIT: valid=%v prefix=%g measure=%v denominator=%v", INVALID_UNIT.Valid(), INVALID_UNIT.GetPrefix(), INVALID_UNIT.GetMeasure(), INVALID_UNIT.GetUnitDenominator())
	}
	convert, out := GetUnitPrefixFactor(NewUnit("unknown"), Base)
	if convert != nil || out.Valid() || out.GetMeasure() != InvalidMeasure {
		t.Errorf("invalid input returned converter=%v, valid output=%v (%q)", convert != nil, out.Valid(), out.Short())
	}
	// Sentinel initialization must not depend on tightening the existing measure parser.
	if u := NewUnit("foobar"); !u.Valid() || u.GetMeasure() != Flops {
		t.Errorf("unrelated permissive parser behavior changed")
	}
}

func TestUnitConverters_InvalidUnitsAndPrefixes(t *testing.T) {
	invalidPrefix := NewUnit("B")
	invalidPrefix.SetPrefix(InvalidPrefix)
	invalidTemperatureIn := NewUnit("degC")
	invalidTemperatureIn.SetPrefix(InvalidPrefix)
	invalidTemperatureOut := NewUnit("degF")
	invalidTemperatureOut.SetPrefix(InvalidPrefix)
	invalidMeasure := &unit{prefix: Base, measure: InvalidMeasure, divMeasure: InvalidMeasure}
	cases := []struct {
		name          string
		input, output Unit
	}{
		{"both invalid", NewUnit("unknown"), NewUnit("unknown")},
		{"input invalid", NewUnit("unknown"), NewUnit("B")},
		{"output invalid", NewUnit("B"), NewUnit("unknown")},
		{"input prefix invalid", invalidPrefix, NewUnit("B")},
		{"output prefix invalid", NewUnit("B"), invalidPrefix},
		{"input measure invalid", invalidMeasure, NewUnit("B")},
		{"output measure invalid", NewUnit("B"), invalidMeasure},
		{"Celsius input prefix invalid", invalidTemperatureIn, NewUnit("degF")},
		{"Fahrenheit output prefix invalid", NewUnit("degC"), invalidTemperatureOut},
	}
	for _, tc := range cases {
		convert, err := GetUnitUnitFactor(tc.input, tc.output)
		if err == nil || convert != nil {
			t.Errorf("%s: err=%v nonnil_converter=%v", tc.name, err, convert != nil)
		}
	}
	prefixCases := []struct {
		name   string
		input  Unit
		output Prefix
	}{
		{"input prefix invalid", invalidPrefix, Base},
		{"input measure invalid", invalidMeasure, Base},
		{"input fully invalid", NewUnit("unknown"), Base},
		{"output prefix invalid", NewUnit("B"), InvalidPrefix},
	}
	for _, tc := range prefixCases {
		convert, out := GetUnitPrefixFactor(tc.input, tc.output)
		if convert != nil || out.Valid() {
			t.Errorf("%s: nonnil_converter=%v output_valid=%v (%q)", tc.name, convert != nil, out.Valid(), out.Short())
		}
	}
	for _, input := range []Unit{NewUnit("B"), NewUnit("unknown"), invalidMeasure, invalidPrefix} {
		convert, out := GetUnitPrefixStringFactor(input, "unknown")
		if convert != nil || out.Valid() {
			t.Errorf("string output prefix: nonnil_converter=%v output_valid=%v", convert != nil, out.Valid())
		}
	}
	for _, tc := range []struct{ input, output string }{{"unknown", "M"}, {"B", "unknown"}, {"unknown", "unknown"}} {
		convert, out := GetUnitStringPrefixStringFactor(tc.input, tc.output)
		if convert != nil || out.Valid() {
			t.Errorf("all-string %q->%q: nonnil_converter=%v output_valid=%v", tc.input, tc.output, convert != nil, out.Valid())
		}
	}
}

func TestUnitConverters_ValidControls(t *testing.T) {
	for _, tc := range []struct {
		input, output string
		value, want   float64
	}{
		{"KB/s", "MB/s", 1000, 1}, {"KiB", "B", 1, 1024}, {"us", "s", 1000000, 1}, {"ns", "s", 1000000000, 1},
		{"degC", "degF", 0, 32}, {"degC", "degF", 100, 212}, {"degF", "degC", 32, 0}, {"degF", "degC", 212, 100},
	} {
		convert, err := GetUnitUnitFactor(NewUnit(tc.input), NewUnit(tc.output))
		if err != nil || convert == nil {
			t.Errorf("valid unit %q->%q rejected: %v", tc.input, tc.output, err)
			continue
		}
		value := convert(tc.value)
		if got, ok := value.(float64); !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.want) > 1e-10 {
			t.Errorf("valid unit %q->%q: got %v (%T), want %g", tc.input, tc.output, value, value, tc.want)
		}
	}
	for _, tc := range []struct {
		input, prefix, output string
		value, want           float64
	}{
		{"KB/s", "M", "MB/s", 1000, 1}, {"B", "Ki", "KiB", 1024, 1}, {"s", "u", "us", 1, 1000000}, {"s", "n", "ns", 1, 1000000000},
	} {
		convert, out := GetUnitPrefixStringFactor(NewUnit(tc.input), tc.prefix)
		if convert == nil || !out.Valid() || out.Short() != tc.output {
			t.Errorf("valid unit-prefix %q->%q rejected or changed", tc.input, tc.prefix)
			continue
		}
		value := convert(tc.value)
		if got, ok := value.(float64); !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("unit-prefix value=%v, want %g", value, tc.want)
		}
		convert, out = GetUnitStringPrefixStringFactor(tc.input, tc.prefix)
		if convert == nil || !out.Valid() || out.Short() != tc.output {
			t.Errorf("valid string wrapper %q->%q changed", tc.input, tc.prefix)
			continue
		}
		value = convert(tc.value)
		if got, ok := value.(float64); !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("string wrapper value=%v, want %g", value, tc.want)
		}
	}
	for _, s := range []string{"B", "KB/s", "us"} {
		input := NewUnit(s)
		convert, err := GetUnitUnitFactor(input, NewUnit(s))
		if err != nil || convert == nil {
			t.Errorf("identity rejected for %q", s)
			continue
		}
		for _, value := range []any{int64(math.MaxInt64), uint64(math.MaxUint64), int64(9007199254740993)} {
			if got := convert(value); !reflect.DeepEqual(got, value) {
				t.Errorf("identity %q lost %T(%v): %T(%v)", s, value, value, got, got)
			}
		}
	}
	for _, tc := range []struct{ input, output string }{{"B", "W"}, {"B/s", "B"}} {
		if _, err := GetUnitUnitFactor(NewUnit(tc.input), NewUnit(tc.output)); err == nil {
			t.Errorf("dimension mismatch %q->%q accepted", tc.input, tc.output)
		}
	}
}
