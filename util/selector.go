// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"encoding/json"
	"errors"
)

// SelectorElement represents a selector that can be either a single string, an array of strings, or a wildcard.
// It supports special JSON marshaling/unmarshaling behavior:
// - A single string (e.g., "value") is stored in the String field
// - An array of strings (e.g., ["val1", "val2"]) is stored in the Group field
// - The wildcard "*" is represented by setting Any to true
type SelectorElement struct {
	String string
	Group  []string
	Any    bool
}

// UnmarshalJSON implements json.Unmarshaler for SelectorElement.
// It handles three formats:
// - A JSON string (converted to String field, or Any if "*")
// - A JSON array (converted to Group field)
// - Any other format returns an error
func (se *SelectorElement) UnmarshalJSON(input []byte) error {
	if input[0] == '"' {
		if err := json.Unmarshal(input, &se.String); err != nil {
			return err
		}

		if se.String == "*" {
			se.Any = true
			se.String = ""
		}

		return nil
	}

	if input[0] == '[' {
		return json.Unmarshal(input, &se.Group)
	}

	return errors.New("the Go SelectorElement type can only be a string or an array of strings")
}

// MarshalJSON implements json.Marshaler for SelectorElement.
// It converts the selector back to JSON:
// - Any=true becomes "*"
// - String field becomes a JSON string
// - Group field becomes a JSON array
func (se *SelectorElement) MarshalJSON() ([]byte, error) {
	if se.Any {
		return []byte("\"*\""), nil
	}

	if se.String != "" {
		return json.Marshal(se.String)
	}

	if se.Group != nil {
		return json.Marshal(se.Group)
	}

	return nil, errors.New("a Go Selector must be a non-empty string or a non-empty slice of strings")
}

// Selector is a slice of SelectorElements used for matching and filtering.
type Selector []SelectorElement
