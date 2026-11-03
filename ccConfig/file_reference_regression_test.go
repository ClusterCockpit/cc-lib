// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ccconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHyphenatedPackageFileReferencesAreLoaded(t *testing.T) {
	root := t.TempDir()
	want := `{"probe":"external-config"}`
	external := filepath.Join(root, "external.json")
	if err := os.WriteFile(external, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"ui-file": external, "metric-store-file": external, "metric-store-external-file": external})
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "config.json")
	if err := os.WriteFile(config, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	Init(config)
	t.Cleanup(Reset)
	for _, name := range []string{"ui", "metric-store", "metric-store-external"} {
		if got := string(GetPackageConfig(name)); got != want {
			t.Errorf("package %q loaded %q; want %q", name, got, want)
		}
	}
}
