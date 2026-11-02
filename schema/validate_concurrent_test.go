// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package schema

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Archive data and statistics can be loaded concurrently by separate requests.
// Exercise all schema kinds, including nested embedded references, with separate
// readers so the only state shared by validations belongs to the implementation.
func TestValidate_Concurrent(t *testing.T) {
	metric := json.RawMessage(`{"node":{"unit":{"base":"B/s"},"timestep":10,"series":[{"hostname":"demo001","statistics":{"avg":2,"min":1,"max":3},"data":[1,2,3]}]}}`)
	data := make(map[string]json.RawMessage)
	for _, name := range []string{"cpu_user", "cpu_load", "mem_used", "flops_any", "mem_bw", "net_bw"} {
		data[name] = metric
	}
	// The deployed v2.12 schema requires a filesystem with read/write metrics.
	data["filesystems"] = json.RawMessage(`[{"name":"scratch","type":"nfs","read_bw":` + string(metric) + `,"write_bw":` + string(metric) + `}]`)

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		kind  Kind
		data  string
		valid bool
	}{
		{"valid job data with embedded references", Data, string(raw), true},
		{"job data missing required metrics", Data, `{}`, false},
		{"metadata missing required fields", Meta, `{}`, false},
		{"cluster missing required fields", ClusterCfg, `{}`, false},
	}
	const workers = 12
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range 4 {
				for _, tc := range cases {
					err := Validate(tc.kind, strings.NewReader(tc.data))
					if tc.valid {
						if err != nil {
							t.Errorf("%s: %v", tc.name, err)
						}
						continue
					}
					var validationError *jsonschema.ValidationError
					if !errors.As(err, &validationError) {
						t.Errorf("%s: want validation error, got %v", tc.name, err)
					}
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}
