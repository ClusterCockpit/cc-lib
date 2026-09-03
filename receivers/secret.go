// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package receivers

import (
	"fmt"

	"github.com/ClusterCockpit/cc-lib/v2/util"
)

// secretRef names the three places one credential of a receiver instance may
// come from. Value points at the inline configuration value and receives the
// resolved secret.
//
// Receivers are configured as a map of named instances, so no fixed
// environment variable name can address the credential of one particular
// receiver. Each instance therefore names its own sources, through the sibling
// configuration keys "<key>_env" and "<key>_file".
type secretRef struct {
	Key      string // configuration key, used in error messages only
	Value    *string
	EnvName  string
	FilePath string
}

// resolveSecrets resolves each secret in place, in the order given. A secret
// whose named file cannot be read is an error rather than a silent fallback to
// the inline value, so a receiver never connects with a stale credential. The
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

// Fixed environment variable names for the BMC credentials of the redfish and
// ipmi receivers. Unlike the credentials above these are one per receiver
// type, not one per instance, so several instances of the same receiver type
// share them; a per-host client_config entry always takes precedence.
//
// Each also accepts a "_FILE" variant naming a file that holds the value. The
// names are prefixed CC_ because cc-lib is linked into several applications,
// whose environments it must not silently claim names in.
const (
	EnvRedfishUsername = "CC_REDFISH_USERNAME"
	EnvRedfishPassword = "CC_REDFISH_PASSWORD"
	EnvIPMIUsername    = "CC_IPMI_USERNAME"
	EnvIPMIPassword    = "CC_IPMI_PASSWORD"
)

// secretDefaultFromEnv supplies a global default credential from the
// environment when the configuration file does not set one. It leaves a
// configured value untouched, so it needs no new configuration key, and an
// absent variable leaves the default unset for the caller to reject.
func secretDefaultFromEnv(target **string, envVar string) error {
	if *target != nil {
		return nil
	}

	v, err := util.SecretFromEnv(envVar, "")
	if err != nil {
		return fmt.Errorf("resolving %s: %w", envVar, err)
	}
	if v == "" {
		return nil
	}

	*target = &v

	return nil
}
