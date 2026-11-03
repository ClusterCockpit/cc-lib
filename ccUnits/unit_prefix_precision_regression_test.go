// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccunits

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func TestNewUnit_MicroNanoPrefixes(t *testing.T) {
	cases := []struct {
		input       string
		prefix      Prefix
		measure     Measure
		denominator Measure
	}{
		{"us", Micro, Time, InvalidMeasure},
		{"usec", Micro, Time, InvalidMeasure},
		{"useconds", Micro, Time, InvalidMeasure},
		{"ns", Nano, Time, InvalidMeasure},
		{"nsec", Nano, Time, InvalidMeasure},
		{"nseconds", Nano, Time, InvalidMeasure},
		{"us/s", Micro, Time, Time},
		{"ns/s", Nano, Time, Time},
		{"uW", Micro, Watt, InvalidMeasure},
		{"nJ", Nano, Joule, InvalidMeasure},
		{"uHz", Micro, Frequency, InvalidMeasure},
		{"nHz", Nano, Frequency, InvalidMeasure},
	}
	for _, tc := range cases {
		u := NewUnit(tc.input)
		if !u.Valid() || u.GetPrefix() != tc.prefix || u.GetMeasure() != tc.measure || u.GetUnitDenominator() != tc.denominator {
			t.Errorf("NewUnit(%q): valid=%v prefix=%g measure=%v denominator=%v, want %g/%v/%v", tc.input, u.Valid(), u.GetPrefix(), u.GetMeasure(), u.GetUnitDenominator(), tc.prefix, tc.measure, tc.denominator)
		}
	}
	// Micro and nano remain unsupported for the documented non-dividable measures.
	for _, p := range []string{"u", "n"} {
		for _, m := range []string{"B", "Flops", "packets", "events", "cyc", "requests"} {
			for _, d := range []string{"", "/s"} {
				if u := NewUnit(p + m + d); u.Valid() {
					t.Errorf("non-dividable unit %q accepted as %q", p+m+d, u.Short())
				}
			}
		}
	}
	// Characterize every previously supported canonical prefix, measure and denominator.
	prefixes := []struct {
		short string
		value Prefix
	}{
		{"", Base}, {"K", Kilo}, {"M", Mega}, {"G", Giga}, {"T", Tera}, {"P", Peta}, {"E", Exa}, {"Z", Zetta}, {"Y", Yotta},
		{"m", Milli}, {"Ki", Kibi}, {"Mi", Mebi}, {"Gi", Gibi}, {"Ti", Tebi}, {"Pi", Pebi}, {"Ei", Exbi}, {"Zi", Zebi}, {"Yi", Yobi},
	}
	measures := []struct {
		short string
		value Measure
	}{
		{"B", Bytes}, {"Flops", Flops}, {"%", Percentage}, {"degC", TemperatureC}, {"degF", TemperatureF}, {"RPM", Rotation}, {"Hz", Frequency},
		{"s", Time}, {"W", Watt}, {"J", Joule}, {"cyc", Cycles}, {"requests", Requests}, {"packets", Packets}, {"events", Events},
	}
	denominators := append([]struct {
		short string
		value Measure
	}{{"", InvalidMeasure}}, measures...)
	for _, p := range prefixes {
		for _, m := range measures {
			wantPrefix := p.value
			if m.value == Percentage {
				wantPrefix = Base
			}
			if p.value == Milli && (m.value == Bytes || m.value == Flops || m.value == Packets || m.value == Events || m.value == Cycles || m.value == Requests) {
				wantPrefix = Mega
			}
			for _, d := range denominators {
				input := p.short + m.short
				if d.value != InvalidMeasure {
					input += "/" + d.short
				}
				u := NewUnit(input)
				if !u.Valid() || u.GetPrefix() != wantPrefix || u.GetMeasure() != m.value || u.GetUnitDenominator() != d.value {
					t.Errorf("old unit %q changed: valid=%v prefix=%g measure=%v denominator=%v", input, u.Valid(), u.GetPrefix(), u.GetMeasure(), u.GetUnitDenominator())
				}
			}
		}
	}
	for _, s := range []string{"ks", "gs", "ts", "ps", "es", "zs", "ys", "kis", "gis", "tis", "pis", "eis", "zis", "yis"} {
		if !NewUnit(s).Valid() {
			t.Errorf("old prefix alias %q became invalid", s)
		}
	}
}

func TestPrefixConverter_IntegerIdentity(t *testing.T) {
	values := []any{
		int64(9007199254740993), int64(-9007199254740993), uint64(9007199254740993),
		int64(math.MaxInt64), int64(math.MinInt64), uint64(math.MaxUint64),
		int(7), int32(-7), uint(7), uint32(7), float32(1.25), float64(-1.25),
	}
	if strconv.IntSize == 64 {
		largeSigned := int64(9007199254740993)
		largeUnsigned := uint64(9007199254740993)
		values = append(values, int(largeSigned), uint(largeUnsigned))
	}
	for _, p := range []Prefix{Base, Kilo, Kibi, Mega, Milli, Micro, Nano} {
		convert := GetPrefixPrefixFactor(p, p)
		for _, value := range values {
			if got := convert(value); !reflect.DeepEqual(got, value) {
				t.Errorf("identity at prefix %g: %T(%v) -> %T(%v)", p, value, value, got, got)
			}
		}
	}
	// Non-identity type preservation and existing truncation remain unchanged.
	convert := GetPrefixPrefixFactor(Kilo, Base)
	for _, value := range []any{int(2), int32(2), int64(2), uint(2), uint32(2), uint64(2), float32(2), float64(2)} {
		got := convert(value)
		if reflect.TypeOf(got) != reflect.TypeOf(value) || reflect.ValueOf(got).Convert(reflect.TypeOf(float64(0))).Float() != 2000 {
			t.Errorf("non-identity conversion changed for %T", value)
		}
	}
	if got := GetPrefixPrefixFactor(Base, Kilo)(int64(1500)); got != int64(1) {
		t.Errorf("integer truncation changed: %v", got)
	}
	for _, value := range []any{"unchanged", int8(7), uint16(7), nil} {
		if got := convert(value); !reflect.DeepEqual(got, value) {
			t.Errorf("unsupported value changed: %T", value)
		}
	}
	// An invalid prefix pair still follows the existing (unfixed) NaN behavior.
	if got := GetPrefixPrefixFactor(InvalidPrefix, InvalidPrefix)(float64(1)).(float64); !math.IsNaN(got) {
		t.Errorf("unrelated invalid-prefix behavior changed: %g", got)
	}
}
