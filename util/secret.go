// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"os"
	"strings"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// EnvFileSuffix is appended to an environment variable name to obtain the name
// of the variable holding the path of a file that contains the secret.
const EnvFileSuffix = "_FILE"

// SecretFromEnv resolves a secret from the environment or the configuration,
// in this order of precedence:
//
//  1. the environment variable envVar, if set and non-empty
//  2. the contents of the file named by the environment variable
//     envVar+EnvFileSuffix, if that variable is set and non-empty
//  3. configValue, the value read from the configuration file
//
// Step 2 lets a deployment inject a secret from a Docker or Kubernetes secret
// mount, or from systemd LoadCredential, without ever placing it in the process
// environment or in a configuration file.
//
// An empty environment variable counts as unset. File contents are trimmed of
// leading and trailing whitespace so that secret files may end in a newline; a
// secret with significant leading or trailing whitespace can therefore not be
// supplied through a file.
//
// An error is returned only when envVar+EnvFileSuffix is set but the file
// cannot be read or holds no non-whitespace characters. An operator who sets
// that variable intends the file to win, so falling back to configValue there
// would silently start the process with a stale credential.
//
// envVar must not itself end in EnvFileSuffix, as it would then shadow the file
// variable of another secret. An empty envVar disables both env sources and
// returns configValue unchanged.
func SecretFromEnv(envVar, configValue string) (string, error) {
	if envVar == "" {
		return configValue, nil
	}

	if v := os.Getenv(envVar); v != "" {
		cclog.Debugf("using secret from environment variable %s", envVar)
		return v, nil
	}

	fileVar := envVar + EnvFileSuffix
	if path := os.Getenv(fileVar); path != "" {
		secret, err := readSecretFile(path)
		if err != nil {
			return "", fmt.Errorf("%s: %w", fileVar, err)
		}
		cclog.Debugf("using secret from the file named by %s", fileVar)
		return secret, nil
	}

	return configValue, nil
}

// SecretFromConfig resolves a secret for one instance of a repeated
// configuration section (one sink, one receiver), where no fixed environment
// variable name exists. The instance names the sources itself, via sibling
// configuration keys. Precedence:
//
//  1. the environment variable envName, if envName is non-empty and that
//     variable is set and non-empty
//  2. the contents of filePath, if filePath is non-empty
//  3. value, the secret configured inline
//
// An empty envName or filePath means that source is not configured. As in
// SecretFromEnv, file contents are whitespace-trimmed, and an unreadable or
// effectively empty file is an error rather than a silent fallback.
func SecretFromConfig(value, envName, filePath string) (string, error) {
	if envName != "" {
		if v := os.Getenv(envName); v != "" {
			cclog.Debugf("using secret from environment variable %s", envName)
			return v, nil
		}
	}

	if filePath != "" {
		secret, err := readSecretFile(filePath)
		if err != nil {
			return "", err
		}
		cclog.Debugf("using secret from the file %s", filePath)
		return secret, nil
	}

	return value, nil
}

// readSecretFile reads and trims a secret file. It never includes the file's
// contents in an error, only its path.
func readSecretFile(path string) (string, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}

	secret := strings.TrimSpace(string(buf))
	if secret == "" {
		return "", fmt.Errorf("secret file %s contains no non-whitespace characters", path)
	}

	return secret, nil
}
