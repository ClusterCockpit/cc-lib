// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package ccconfig

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

var keys map[string]json.RawMessage

func Init(filename string) {
	raw, err := os.ReadFile(filename)
	jkeys := make(map[string]json.RawMessage)

	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("CONFIG ERROR: %v", err)
		}
	} else {
		dec := json.NewDecoder(bytes.NewReader(raw))
		if err := dec.Decode(&jkeys); err != nil {
			log.Fatalf("could not decode: %v", err)
		}
	}

	keys = make(map[string]json.RawMessage)

	for k, v := range jkeys {
		s := strings.Split(k, "-")
		if len(s) == 2 && s[1] == "file" {
			var filename string
			err := json.Unmarshal(v, &filename)
			if err != nil {
				log.Fatalln("error:", err)
			}
			b, err := os.ReadFile(filename)
			if err != nil {
				cclog.ComponentError("ccConfig", err.Error())
			}

			keys[s[0]] = b
		} else {
			keys[k] = jkeys[k]
		}
	}
}

func GetPackageConfig(key string) json.RawMessage {
	if val, ok := keys[key]; ok {
		return val
	} else {
		cclog.Errorf("CONFIG ERROR: Key %s not found", key)
		return nil
	}
}
