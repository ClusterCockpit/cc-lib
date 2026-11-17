// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package resampler

import (
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"testing"
)

func TestLTTB_SingleTargetUsesConsistentResolution(t *testing.T) {
	data := make([]schema.Float, 1000)
	for i := range data {
		data[i] = schema.Float(i)
	}
	got, step, err := LargestTriangleThreeBucket(data, 1, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !((len(got) == 1 && step == 1000) || (len(got) == len(data) && step == 1)) {
		t.Fatalf("inconsistent output: len=%d step=%d", len(got), step)
	}
}

func TestLTTB_PreservesPenultimatePeak(t *testing.T) {
	for _, length := range []int{1000, 1200, 2000} {
		for _, step := range []int64{2, 5, 10} {
			data := make([]schema.Float, length)
			data[length-2] = 100
			got, _, err := LargestTriangleThreeBucket(data, 1, step)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, v := range got {
				if v == 100 {
					found = true
				}
			}
			if !found {
				t.Errorf("penultimate peak lost: length=%d step=%d", length, step)
			}
		}
	}
}
