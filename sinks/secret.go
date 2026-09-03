// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sinks

import (
	"fmt"

	"github.com/ClusterCockpit/cc-lib/v2/util"
)

// secretRef names the three places one credential of a sink instance may come
// from. Value points at the inline configuration value and receives the
// resolved secret.
//
// Sinks are configured as a map of named instances, so no fixed environment
// variable name can address the credential of one particular sink. Each
// instance therefore names its own sources, through the sibling configuration
// keys "<key>_env" and "<key>_file".
type secretRef struct {
	Key      string // configuration key, used in error messages only
	Value    *string
	EnvName  string
	FilePath string
}

// resolveSecrets resolves each secret in place, in the order given. A secret
// whose named file cannot be read is an error rather than a silent fallback to
// the inline value, so a sink never connects with a stale credential. The
// resolved values are never logged.
func resolveSecrets(secrets ...secretRef) error {
	for _, s := range secrets {
		v, err := util.SecretFromConfig(*s.Value, s.EnvName, s.FilePath)
		if err != nil {
			return fmt.Errorf("resolving %q: %w", s.Key, err)
		}
		*s.Value = v
	}

	return nil
}
