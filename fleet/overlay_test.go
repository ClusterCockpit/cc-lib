// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

type testConf struct {
	URL      string              `json:"url"`
	Token    string              `json:"token,omitempty"`
	PidFile  string              `json:"pidFile"`
	Interval int                 `json:"interval"`
	Filter   string              `json:"filter"`
	Server   string              `json:"server"`
	Subject  string              `json:"subject"`
	Devices  map[string][]string `json:"devices"`
	Plain    bool
	Skipped  string `json:"-"`
	internal int
}

const testLocal = `{
	"url": "http://local:8080",
	"token": "local-token",
	"interval": 30,
	"filter": "^local",
	"server": "nats-local",
	"devices": {"^a": ["00:01"], "^b": ["00:02"]}
}`

func newTestOverlay() *Overlay[testConf] {
	return &Overlay[testConf]{
		Defaults: func() testConf {
			return testConf{PidFile: "/run/test.pid", Interval: 60, Subject: "jobs", Devices: map[string][]string{}}
		},
		Local:       []byte(testLocal),
		LocalOnly:   []string{"url", "token", "pidFile"},
		StartupOnly: []string{"server"},
		Validate: func(c *testConf) error {
			if c.Interval < 1 {
				c.Interval = 1
			}
			if c.Filter == "(" {
				return errors.New("invalid filter")
			}
			return nil
		},
	}
}

func mustBuild(t *testing.T, o *Overlay[testConf], blob string) testConf {
	t.Helper()
	conf, err := o.Build([]byte(blob))
	if err != nil {
		t.Fatalf("Build(%s): %v", blob, err)
	}
	return conf
}

func TestOverlayBuild_LocalOnly(t *testing.T) {
	conf := mustBuild(t, newTestOverlay(), "")
	want := testConf{
		URL: "http://local:8080", Token: "local-token", PidFile: "/run/test.pid", Interval: 30,
		Filter: "^local", Server: "nats-local", Subject: "jobs",
		Devices: map[string][]string{"^a": {"00:01"}, "^b": {"00:02"}},
	}
	if !reflect.DeepEqual(conf, want) {
		t.Errorf("got %+v, want %+v", conf, want)
	}
}

func TestOverlayBuild_Merge(t *testing.T) {
	conf := mustBuild(t, newTestOverlay(), `{
		"filter": "^fleet",
		"devices": {"^b": ["00:03", "00:04"], "^c": ["00:05"]},
		"log-level": "debug"
	}`)

	if conf.Filter != "^fleet" {
		t.Errorf("filter = %q, want fleet value", conf.Filter)
	}
	if conf.Interval != 30 {
		t.Errorf("interval = %d, want local value 30", conf.Interval)
	}
	// Maps merge key by key, arrays are replaced wholesale.
	wantDevices := map[string][]string{"^a": {"00:01"}, "^b": {"00:03", "00:04"}, "^c": {"00:05"}}
	if !reflect.DeepEqual(conf.Devices, wantDevices) {
		t.Errorf("devices = %v, want %v", conf.Devices, wantDevices)
	}
}

func TestOverlayBuild_LocalOnlyKeysIgnored(t *testing.T) {
	conf := mustBuild(t, newTestOverlay(), `{
		"url": "http://evil:8080",
		"token": "fleet-token",
		"pidFile": "/tmp/x.pid"
	}`)
	if conf.URL != "http://local:8080" || conf.Token != "local-token" || conf.PidFile != "/run/test.pid" {
		t.Errorf("local-only keys overridden by fleet: %+v", conf)
	}
}

func TestOverlayBuild_Errors(t *testing.T) {
	tests := []struct {
		name    string
		overlay func() *Overlay[testConf]
		blob    string
	}{
		{"validate", newTestOverlay, `{"filter": "("}`},
		{"wrong type", newTestOverlay, `{"interval": "soon"}`},
		{"not an object", newTestOverlay, `[1, 2]`},
		{"bad local", func() *Overlay[testConf] {
			o := newTestOverlay()
			o.Local = []byte(`{`)
			return o
		}, ""},
		{"unknown local-only key", func() *Overlay[testConf] {
			o := newTestOverlay()
			o.LocalOnly = append(o.LocalOnly, "nope")
			return o
		}, ""},
		{"unknown startup-only key", func() *Overlay[testConf] {
			o := newTestOverlay()
			o.StartupOnly = []string{"Skipped"}
			return o
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.overlay().Build([]byte(tt.blob)); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestOverlayBuild_NotAStruct(t *testing.T) {
	o := &Overlay[map[string]any]{}
	if _, err := o.Build(nil); err == nil {
		t.Error("expected error for non-struct type")
	}
}

func TestOverlayBuild_Validate(t *testing.T) {
	conf := mustBuild(t, newTestOverlay(), `{"interval": 0}`)
	if conf.Interval != 1 {
		t.Errorf("interval = %d, want normalized 1", conf.Interval)
	}
}

func TestOverlayApply_Startup(t *testing.T) {
	o := newTestOverlay()
	current := mustBuild(t, o, "")

	res, err := o.Apply(current, []byte(`{"server": "nats-fleet", "filter": "^fleet", "Plain": true}`), true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Config.Server != "nats-fleet" {
		t.Errorf("server = %q, want fleet value at startup", res.Config.Server)
	}
	if len(res.RestartRequired) != 0 {
		t.Errorf("RestartRequired = %v, want none at startup", res.RestartRequired)
	}
	if want := []string{"filter", "server", "Plain"}; !slices.Equal(res.Changed, want) {
		t.Errorf("Changed = %v, want %v", res.Changed, want)
	}
	if current.Server != "nats-local" {
		t.Errorf("Apply modified current")
	}
}

func TestOverlayApply_Runtime(t *testing.T) {
	o := newTestOverlay()
	current := mustBuild(t, o, "")

	res, err := o.Apply(current, []byte(`{"server": "nats-fleet", "subject": "fleetjobs"}`), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Config.Server != "nats-local" {
		t.Errorf("server = %q, want running value at runtime", res.Config.Server)
	}
	if !slices.Equal(res.RestartRequired, []string{"server"}) {
		t.Errorf("RestartRequired = %v, want [server]", res.RestartRequired)
	}
	if res.Config.Subject != "fleetjobs" {
		t.Errorf("subject = %q, want fleet value", res.Config.Subject)
	}
	if !slices.Equal(res.Changed, []string{"subject"}) {
		t.Errorf("Changed = %v, want [subject]", res.Changed)
	}
}

func TestOverlayApply_EmptyRevertsToLocal(t *testing.T) {
	o := newTestOverlay()
	current := mustBuild(t, o, `{"filter": "^fleet", "interval": 10}`)

	res, err := o.Apply(current, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Config.Filter != "^local" || res.Config.Interval != 30 {
		t.Errorf("got %+v, want local configuration", res.Config)
	}
	if want := []string{"interval", "filter"}; !slices.Equal(res.Changed, want) {
		t.Errorf("Changed = %v, want %v", res.Changed, want)
	}
}

func TestOverlayApply_ErrorReturnsNoResult(t *testing.T) {
	o := newTestOverlay()
	current := mustBuild(t, o, "")
	if _, err := o.Apply(current, []byte(`{"filter": "("}`), false); err == nil {
		t.Error("expected error")
	}
}
