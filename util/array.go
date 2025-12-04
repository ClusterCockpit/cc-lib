// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import "slices"

// Contains checks if an item exists in a slice of comparable items.
// It is a generic wrapper around slices.Contains that works with any comparable type T.
func Contains[T comparable](items []T, item T) bool {
	return slices.Contains(items, item)
}
