// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccunits

import (
	"math"
	"testing"
)

func TestUnitDenominatorValidation(t *testing.T) {
	for _, input := range []string{
		"B/noSuchMeasure", "B/ms", "B/", "B//s", "B/s/J", "B/s/",
		"KB/s/s", "GHz//B", "degC/ms", "degF/s/J",
	} {
		unit := NewUnit(input)
		if unit.Valid() || unit != INVALID_UNIT {
			t.Errorf("NewUnit(%q) = %q valid=%v; want INVALID_UNIT", input, unit.Short(), unit.Valid())
		}
		convert, output := GetUnitStringPrefixStringFactor(input, "K")
		if convert != nil || output.Valid() {
			t.Errorf("invalid denominator %q yielded a prefix converter or valid output", input)
		}
		if _, err := GetUnitUnitFactor(unit, NewUnit("B")); err == nil {
			t.Errorf("invalid denominator %q accepted by Unit-to-Unit API", input)
		}
	}
	for _, tc := range []struct {
		input       string
		measure     Measure
		denominator Measure
	}{
		{"B", Bytes, InvalidMeasure}, {"KB", Bytes, InvalidMeasure},
		{"B/s", Bytes, Time}, {"Flops/B", Flops, Bytes},
		{"W/B", Watt, Bytes}, {"degC", TemperatureC, InvalidMeasure},
		{"degC/s", TemperatureC, Time}, {"degF/J", TemperatureF, Joule},
	} {
		unit := NewUnit(tc.input)
		if !unit.Valid() || unit.GetMeasure() != tc.measure || unit.GetUnitDenominator() != tc.denominator {
			t.Errorf("valid optional/base denominator changed for %q: %q", tc.input, unit.Short())
		}
	}
}

func TestUnitTemperatureDenominatorCompatibility(t *testing.T) {
	for _, tc := range []struct{ in, out string }{
		{"degC/s", "degF"}, {"degC", "degF/s"}, {"degC/s", "degF/J"},
		{"degF/s", "degC"}, {"degF", "degC/s"}, {"degF/J", "degC/s"},
		{"degC/s", "degC"}, {"degF/s", "degF"}, {"KB/s", "MB/J"},
	} {
		in, out := NewUnit(tc.in), NewUnit(tc.out)
		if !in.Valid() || !out.Valid() {
			t.Fatalf("invalid fixture %q -> %q", tc.in, tc.out)
		}
		if _, err := GetUnitUnitFactor(in, out); err == nil {
			t.Errorf("incompatible denominators accepted: %q -> %q", tc.in, tc.out)
		}
	}
	for _, tc := range []struct {
		in, out     string
		value, want float64
	}{
		{"degC", "degF", 0, 32}, {"degC", "degF", 100, 212},
		{"degF", "degC", 32, 0}, {"degF", "degC", 212, 100},
	} {
		convert, err := GetUnitUnitFactor(NewUnit(tc.in), NewUnit(tc.out))
		if err != nil || convert == nil {
			t.Fatalf("valid temperature converter %q -> %q: %v", tc.in, tc.out, err)
		}
		got, ok := convert(tc.value).(float64)
		if !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("temperature %q -> %q: got=%v want=%v", tc.in, tc.out, got, tc.want)
		}
	}
}

func TestUnitDimensionCompatibility(t *testing.T) {
	for _, tc := range []struct {
		in, out     string
		value, want float64
	}{
		{"KB/s", "MB/s", 2000, 2}, {"Flops/B", "MFlops/B", 2000000, 2},
		{"us", "ns", 1, 1000}, {"ns", "us", 1000, 1},
	} {
		convert, err := GetUnitUnitFactor(NewUnit(tc.in), NewUnit(tc.out))
		if err != nil || convert == nil {
			t.Fatalf("valid converter %q -> %q: %v", tc.in, tc.out, err)
		}
		got, ok := convert(tc.value).(float64)
		if !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%q -> %q: got=%v want=%v", tc.in, tc.out, got, tc.want)
		}
	}
	identity, err := GetUnitUnitFactor(NewUnit("B/s"), NewUnit("B/s"))
	if err != nil || identity == nil {
		t.Fatalf("identity: %v", err)
	}
	largeSigned := int64(9007199254740993)
	largeUnsigned := uint64(18446744073709551615)
	if identity(largeSigned) != largeSigned || identity(largeUnsigned) != largeUnsigned {
		t.Fatal("integer identity precision lost")
	}
	if INVALID_UNIT.Valid() {
		t.Fatal("invalid sentinel became valid")
	}
	if convert, out := GetUnitPrefixFactor(INVALID_UNIT, Kilo); convert != nil || out.Valid() {
		t.Fatal("invalid-unit guard lost")
	}

	// Characterize unresolved legacy behavior, not a new temperature-rate or prefix contract.
	// This limited fix must not silently change affine temperature/prefix semantics.
	for _, tc := range []struct {
		in, out         string
		value, observed float64
	}{
		{"KdegC", "degF", 1, 33.8}, {"degC", "KdegF", 1, 33.8},
		{"degC/s", "degF/s", 1, 33.8}, {"degF/s", "degC/s", 33.8, 1},
	} {
		convert, err := GetUnitUnitFactor(NewUnit(tc.in), NewUnit(tc.out))
		if err != nil || convert == nil {
			t.Fatalf("legacy characterization %q -> %q: %v", tc.in, tc.out, err)
		}
		got, ok := convert(tc.value).(float64)
		if !ok || math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-tc.observed) > 1e-12 {
			t.Errorf("legacy behavior changed %q -> %q: got=%v observed=%v", tc.in, tc.out, got, tc.observed)
		}
	}
}
