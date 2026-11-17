// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"fmt"
	"strings"
	"testing"
)

func TestJobGoString_UsesDatabaseIDValue(t *testing.T) {
	for _, id := range []int64{0, 42, 9007199254740993} {
		job := Job{ID: &id, JobID: 123, StartTime: 1700000000}
		prefix := fmt.Sprintf("Job{ID:%d,", id)
		if got := job.GoString(); !strings.HasPrefix(got, prefix) {
			t.Errorf("ID pointer formatted instead of value: %s", got)
		}
		if got := fmt.Sprintf("%#v", job); !strings.HasPrefix(got, prefix) {
			t.Errorf("GoStringer ID differs: %s", got)
		}
	}
	if got := (Job{}).GoString(); !strings.HasPrefix(got, "Job{ID:0,") {
		t.Errorf("nil-ID convention changed: %s", got)
	}
}
