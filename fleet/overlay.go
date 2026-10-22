// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// Overlay merges a fleet configuration onto an application's local
// configuration. T is a struct whose top-level fields are addressed by their
// JSON keys.
//
// The effective configuration is built in layers: Defaults, then the local
// configuration file, then the fleet configuration. Decoding each layer into
// the same value means JSON objects decoded into maps merge key by key, while
// arrays and scalars are replaced.
//
// Which keys a fleet configuration may set is the application's policy:
// LocalOnly keys always come from the local file, StartupOnly keys apply at
// startup but keep their running value afterwards.
type Overlay[T any] struct {
	// Defaults returns the built-in configuration. It is called once per
	// layer, so it must return values that share no maps or slices. Nil
	// starts from the zero value.
	Defaults func() T
	// Local is the raw local configuration file. Empty means defaults only.
	Local []byte
	// LocalOnly lists keys a fleet configuration cannot set, for example the
	// settings needed to reach cc-backend or host-local paths.
	LocalOnly []string
	// StartupOnly lists keys that only take effect at startup, for example
	// connection settings that are used once.
	StartupOnly []string
	// Validate checks and may normalize a built configuration. Optional.
	Validate func(*T) error
}

// Result is a configuration built by [Overlay.Apply].
type Result[T any] struct {
	Config T
	// Changed lists the keys whose value differs from the current
	// configuration.
	Changed []string
	// RestartRequired lists the StartupOnly keys whose fleet value differs
	// from the running value. Only set at runtime; Config keeps the running
	// values.
	RestartRequired []string
}

// Build returns the configuration for the fleet configuration blob. An empty
// blob means no fleet configuration applies, so the result is the local
// configuration. Keys of the blob that are local-only are warned about,
// unknown keys are logged at debug level: the top-level defaults of the fleet
// configuration tree apply to all service types.
func (o *Overlay[T]) Build(blob []byte) (T, error) {
	var zero T
	if err := o.check(); err != nil {
		return zero, err
	}

	conf, err := o.decode(blob)
	if err != nil {
		return zero, err
	}

	if len(blob) > 0 {
		o.reportKeys(blob)

		local, err := o.decode(nil)
		if err != nil {
			return zero, err
		}
		for _, key := range o.LocalOnly {
			field(&conf, key).Set(field(&local, key))
		}
	}

	if o.Validate != nil {
		if err := o.Validate(&conf); err != nil {
			return zero, err
		}
	}
	return conf, nil
}

// Apply builds the configuration for blob and compares it with current, which
// is not modified. At startup all keys apply. Otherwise StartupOnly keys keep
// their value from current and the ones that would have changed are reported
// in RestartRequired.
//
// On error the application should keep running with current. Call
// [Result.Log] once the result is committed.
func (o *Overlay[T]) Apply(current T, blob []byte, startup bool) (Result[T], error) {
	conf, err := o.Build(blob)
	if err != nil {
		return Result[T]{}, err
	}

	res := Result[T]{}
	if !startup {
		for _, key := range o.StartupOnly {
			if !equal(&conf, &current, key) {
				res.RestartRequired = append(res.RestartRequired, key)
			}
			field(&conf, key).Set(field(&current, key))
		}
	}

	for _, key := range keys(reflect.TypeFor[T]()) {
		if !equal(&conf, &current, key) {
			res.Changed = append(res.Changed, key)
		}
	}

	res.Config = conf
	return res, nil
}

// Log reports the changed keys and the keys that need a restart.
func (r Result[T]) Log() {
	if len(r.Changed) > 0 {
		cclog.ComponentInfof(component, "applied configuration, changed keys: %v", r.Changed)
	} else {
		cclog.ComponentDebugf(component, "applied configuration, no keys changed")
	}
	if len(r.RestartRequired) > 0 {
		cclog.ComponentWarnf(component, "configuration keys %v changed, but only take effect after a restart", r.RestartRequired)
	}
}

// check verifies that T is a struct and that the key lists name its fields.
func (o *Overlay[T]) check() error {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("fleet: overlay type %s is not a struct", t)
	}
	known := keys(t)
	for _, key := range slices.Concat(o.LocalOnly, o.StartupOnly) {
		if !slices.Contains(known, key) {
			return fmt.Errorf("fleet: overlay key %q is not a field of %s", key, t)
		}
	}
	return nil
}

// decode layers Defaults, Local and blob into a new value.
func (o *Overlay[T]) decode(blob []byte) (T, error) {
	var conf T
	if o.Defaults != nil {
		conf = o.Defaults()
	}
	if len(o.Local) > 0 {
		if err := json.Unmarshal(o.Local, &conf); err != nil {
			return conf, fmt.Errorf("fleet: decoding local configuration: %w", err)
		}
	}
	if len(blob) > 0 {
		if err := json.Unmarshal(blob, &conf); err != nil {
			return conf, fmt.Errorf("fleet: decoding fleet configuration: %w", err)
		}
	}
	return conf, nil
}

func (o *Overlay[T]) reportKeys(blob []byte) {
	var fleetKeys map[string]json.RawMessage
	if err := json.Unmarshal(blob, &fleetKeys); err != nil {
		return
	}

	known := keys(reflect.TypeFor[T]())
	for key := range fleetKeys {
		if slices.Contains(o.LocalOnly, key) {
			cclog.ComponentWarnf(component, "ignoring key %q from fleet configuration, it can only be set locally", key)
		} else if !slices.Contains(known, key) {
			cclog.ComponentDebugf(component, "ignoring unknown key %q from fleet configuration", key)
		}
	}
}

// keys returns the JSON keys of the exported fields of t, in field order.
func keys(t reflect.Type) []string {
	var keys []string
	for i := 0; i < t.NumField(); i++ {
		if key := jsonKey(t.Field(i)); key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// jsonKey returns the key encoding/json uses for f, or "" if f is skipped.
func jsonKey(f reflect.StructField) string {
	if !f.IsExported() {
		return ""
	}
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	switch name {
	case "-":
		return ""
	case "":
		return f.Name
	}
	return name
}

// field returns the field of conf for key. The key must be valid (see check).
func field[T any](conf *T, key string) reflect.Value {
	v := reflect.ValueOf(conf).Elem()
	for i := 0; i < v.NumField(); i++ {
		if jsonKey(v.Type().Field(i)) == key {
			return v.Field(i)
		}
	}
	panic(fmt.Sprintf("fleet: unknown overlay key %q", key))
}

func equal[T any](a, b *T, key string) bool {
	return reflect.DeepEqual(field(a, key).Interface(), field(b, key).Interface())
}
