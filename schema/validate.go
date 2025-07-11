// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Kind int

const (
	Meta Kind = iota + 1
	Data
	Config
	ClusterCfg
)

//go:embed schemas/*
var schemaFiles embed.FS

func Validate(k Kind, r io.Reader) error {
	jsonschema.Loaders["embedfs"] = func(s string) (io.ReadCloser, error) {
		f := filepath.Join("schemas", strings.Split(s, "//")[1])
		return schemaFiles.Open(f)
	}
	var s *jsonschema.Schema
	var err error

	switch k {
	case Meta:
		s, err = jsonschema.Compile("embedfs://job-meta.schema.json")
	case Data:
		s, err = jsonschema.Compile("embedfs://job-data.schema.json")
	case ClusterCfg:
		s, err = jsonschema.Compile("embedfs://cluster.schema.json")
	case Config:
		s, err = jsonschema.Compile("embedfs://config.schema.json")
	default:
		return fmt.Errorf("SCHEMA/VALIDATE > unkown schema kind: %#v", k)
	}

	if err != nil {
		cclog.Errorf("Error while compiling json schema for kind '%#v'", k)
		return err
	}

	var v any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		cclog.Warnf("Error while decoding raw json schema: %#v", err)
		return err
	}

	if err = s.Validate(v); err != nil {
		return fmt.Errorf("SCHEMA/VALIDATE > %#v", err)
	}

	return nil
}
