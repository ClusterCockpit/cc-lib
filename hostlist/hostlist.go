// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

// Package hostlist provides functionality to expand compact hostlist specifications
// into individual host names. This is particularly useful for cluster computing
// environments where hosts are often specified using range notation.
//
// The package supports expanding hostlist expressions like:
//   - "n[1-3]" expands to ["n1", "n2", "n3"]
//   - "n[01-03]" expands to ["n01", "n02", "n03"] (preserves zero-padding)
//   - "n[1-2],m[3-4]" expands to ["m3", "m4", "n1", "n2"] (sorted and deduplicated)
//   - "n[1-2]-suffix" expands to ["n1-suffix", "n2-suffix"]
//
// The expansion automatically sorts results alphabetically and removes duplicates.
package hostlist

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var regexPrecompiled struct {
	hl *regexp.Regexp
	indexPrefix,
	indexRange,
	indexSuffix int
}

// init initializes repeatedly used regular expressions
func init() {
	// Create ranges regular expression
	regexNumber := "[[:digit:]]+"
	regexRange := regexNumber + "-" + regexNumber
	regexOptionalNumberOrRange := "(" + regexNumber + ",|" + regexRange + ",)*"
	regexNumberOrRange := "(" + regexNumber + "|" + regexRange + ")"
	regexBraceLeft := "[[]"
	regexBraceRight := "[]]"
	regexRanges := regexBraceLeft +
		regexOptionalNumberOrRange +
		regexNumberOrRange +
		regexBraceRight

	// Create host list regular expression
	regexLeadingDelimiters := "^[,[:space:]]*"
	regexDNSChars := "[a-zA-Z0-9-]+"
	regePrefix := "(?P<prefix>" + regexDNSChars + ")"
	regexOptionalRange := "(?P<range>" + regexRanges + ")?"
	regexOptionalSuffix := "(?P<suffix>" + regexDNSChars + ")?"
	regexPrecompiled.hl = regexp.MustCompile(regexLeadingDelimiters + regePrefix + regexOptionalRange + regexOptionalSuffix)
	regexPrecompiled.indexPrefix = regexPrecompiled.hl.SubexpIndex("prefix")
	regexPrecompiled.indexRange = regexPrecompiled.hl.SubexpIndex("range")
	regexPrecompiled.indexSuffix = regexPrecompiled.hl.SubexpIndex("suffix")
}

// Expand converts a compact hostlist specification into a slice of individual host names.
//
// The function accepts hostlist expressions containing:
//   - Single hosts: "node1"
//   - Numeric ranges in brackets: "node[1-5]"
//   - Multiple ranges or indices: "node[1-3,5,7-9]"
//   - Optional suffixes after ranges: "node[1-3]-ib" for InfiniBand interfaces
//   - Multiple comma-separated expressions: "n[1-2],m[3-4]"
//
// Syntax rules:
//   - Ranges must be specified in brackets using the format [start-end]
//   - Multiple ranges or indices within brackets must be comma-separated
//   - Only one bracketed range specification is allowed per host expression
//   - Range start must be less than or equal to range end
//   - Zero-padding is preserved when start and end have the same width
//   - Valid DNS characters: a-z, A-Z, 0-9, and hyphen (-)
//
// The function automatically:
//   - Sorts the resulting host names alphabetically
//   - Removes duplicate entries
//   - Trims leading/trailing spaces and commas from the input
//
// Parameters:
//   - in: The hostlist specification string to expand
//
// Returns:
//   - result: A sorted slice of unique host names
//   - err: An error if the input is malformed or contains invalid syntax
//
// Examples:
//
//	// Simple range
//	hosts, _ := Expand("n[1-3]")
//	// Returns: []string{"n1", "n2", "n3"}
//
//	// Zero-padded range
//	hosts, _ := Expand("node[01-03]")
//	// Returns: []string{"node01", "node02", "node03"}
//
//	// Multiple ranges and indices
//	hosts, _ := Expand("n[1-2,5,7-8]")
//	// Returns: []string{"n1", "n2", "n5", "n7", "n8"}
//
//	// With suffix
//	hosts, _ := Expand("n[1-2]-ib")
//	// Returns: []string{"n1-ib", "n2-ib"}
//
//	// Multiple host groups
//	hosts, _ := Expand("n[1-2],m[3-4]")
//	// Returns: []string{"m3", "m4", "n1", "n2"}
//
//	// Duplicates are removed
//	hosts, _ := Expand("n1,n1,n2")
//	// Returns: []string{"n1", "n2"}
//
// Error conditions:
//   - Invalid characters (e.g., "@", "$")
//   - Malformed range syntax (e.g., "[1-2-3]")
//   - Decreasing ranges (e.g., "[5-1]")
//   - Invalid bracket nesting or missing brackets
func Expand(in string) (result []string, err error) {
	for len(in) > 0 {
		if v := regexPrecompiled.hl.FindStringSubmatch(in); v != nil {

			// Remove matched part from the input
			lenPrefix := len(v[0])
			in = in[lenPrefix:]

			// matched prefix, range and suffix
			hlPrefix := v[regexPrecompiled.indexPrefix]
			hlRanges := v[regexPrecompiled.indexRange]
			hlSuffix := v[regexPrecompiled.indexSuffix]

			// Single node without ranges
			if hlRanges == "" {
				result = append(result, hlPrefix)
				continue
			}

			// Node with ranges
			if len(hlRanges) > 0 {

				// Remove braces
				hlRanges = hlRanges[1 : len(hlRanges)-1]

				// Split host ranges at ,
				for hlRange := range strings.SplitSeq(hlRanges, ",") {

					// Split host range at -
					RangeStartEnd := strings.Split(hlRange, "-")

					// Range is only a single number
					if len(RangeStartEnd) == 1 {
						result = append(result, hlPrefix+RangeStartEnd[0]+hlSuffix)
						continue
					}

					// Range has a start and an end
					rangeStart := RangeStartEnd[0]
					rangeEnd := RangeStartEnd[1]
					widthRangeStart := len(rangeStart)
					widthRangeEnd := len(rangeEnd)
					iStart, _ := strconv.ParseUint(rangeStart, 10, 64)
					iEnd, _ := strconv.ParseUint(rangeEnd, 10, 64)
					if iStart > iEnd {
						return nil, fmt.Errorf("single range start is greater than end: %s", hlRange)
					}

					// Create print format string for range numbers
					doPadding := widthRangeStart == widthRangeEnd
					if !doPadding && (rangeStart[:1] == "0" || rangeEnd[:1] == "0") {
						return nil, fmt.Errorf("single range without padding is used but start or end use padding: %s", hlRange)
					}
					widthPadding := widthRangeStart
					var formatString string
					if doPadding {
						formatString = "%0" + fmt.Sprint(widthPadding) + "d"
					} else {
						formatString = "%d"
					}
					formatString = hlPrefix + formatString + hlSuffix

					// Add nodes from this range
					for i := iStart; i <= iEnd; i++ {
						result = append(result, fmt.Sprintf(formatString, i))
					}
				}
			}
		} else {
			return nil, fmt.Errorf("not a hostlist: %s", in)
		}
	}

	if result != nil {
		// Sort and uniq
		slices.Sort(result)
		result = slices.Compact(result)
	}

	return
}
