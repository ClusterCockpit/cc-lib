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
	"sort"
	"strconv"
	"strings"
)

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
	// Create ranges regular expression
	// Matches patterns like: [1], [1-5], [1,2], [1-3,5-7], etc.
	// reStNumber: one or more digits
	reStNumber := "[[:digit:]]+"
	// reStRange: two numbers separated by hyphen (e.g., "1-5")
	reStRange := reStNumber + "-" + reStNumber
	// reStOptionalNumberOrRange: zero or more comma-separated numbers or ranges
	reStOptionalNumberOrRange := "(" + reStNumber + ",|" + reStRange + ",)*"
	// reStNumberOrRange: a single number or range (required at end)
	reStNumberOrRange := "(" + reStNumber + "|" + reStRange + ")"
	// Bracket characters (escaped for regex)
	reStBraceLeft := "[[]"
	reStBraceRight := "[]]"
	// Complete range pattern: [optional_items,required_item]
	reStRanges := reStBraceLeft +
		reStOptionalNumberOrRange +
		reStNumberOrRange +
		reStBraceRight
	reRanges := regexp.MustCompile(reStRanges)

	// Create host list regular expression
	// Matches patterns like: prefix[ranges]suffix, where ranges and suffix are optional
	// Valid DNS characters: letters, digits, and hyphens
	reStDNSChars := "[a-zA-Z0-9-]+"
	// Prefix is required and must start at beginning of string
	reStPrefix := "^(" + reStDNSChars + ")"
	// Suffix is optional (e.g., for "-ib" in "node[1-2]-ib")
	reStOptionalSuffix := "(" + reStDNSChars + ")?"
	// Complete pattern: prefix + optional[ranges] + optional_suffix
	re := regexp.MustCompile(reStPrefix + "([[][0-9,-]+[]])?" + reStOptionalSuffix)

	// Remove all delimiters from the input
	in = strings.TrimLeft(in, ", ")

	for len(in) > 0 {
		if v := re.FindStringSubmatch(in); v != nil {

			// Remove matched part from the input
			lenPrefix := len(v[0])
			in = in[lenPrefix:]

			// Remove all delimiters from the input
			in = strings.TrimLeft(in, ", ")

			// matched prefix, range and suffix
			hlPrefix := v[1]
			hlRanges := v[2]
			hlSuffix := v[3]

			// Single node without ranges
			if hlRanges == "" {
				result = append(result, hlPrefix)
				continue
			}

			// Node with ranges
			if v := reRanges.FindStringSubmatch(hlRanges); v != nil {

				// Remove braces
				hlRanges = hlRanges[1 : len(hlRanges)-1]

				// Split host ranges at ,
				for _, hlRange := range strings.Split(hlRanges, ",") {

					// Split host range at -
					RangeStartEnd := strings.Split(hlRange, "-")

					// Range is only a single number
					if len(RangeStartEnd) == 1 {
						result = append(result, hlPrefix+RangeStartEnd[0]+hlSuffix)
						continue
					}

					// Range has a start and an end
					widthRangeStart := len(RangeStartEnd[0])
					widthRangeEnd := len(RangeStartEnd[1])
					iStart, _ := strconv.ParseUint(RangeStartEnd[0], 10, 64)
					iEnd, _ := strconv.ParseUint(RangeStartEnd[1], 10, 64)
					if iStart > iEnd {
						return nil, fmt.Errorf("single range start is greater than end: %s", hlRange)
					}

					// Create print format string for range numbers
					doPadding := widthRangeStart == widthRangeEnd
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
			} else {
				return nil, fmt.Errorf("not at hostlist range: %s", hlRanges)
			}
		} else {
			return nil, fmt.Errorf("not a hostlist: %s", in)
		}
	}

	if result != nil {
		// Sort results alphabetically for consistent output
		sort.Strings(result)

		// Remove duplicates using in-place deduplication algorithm
		// This is more efficient than using a map for large result sets
		previous := 1
		for current := 1; current < len(result); current++ {
			if result[current-1] != result[current] {
				// Found a unique element, copy it to the next position
				if previous != current {
					result[previous] = result[current]
				}
				previous++
			}
		}
		result = result[:previous]
	}

	return
}
