// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fleet

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
)

// cacheIdentity ties a cache file to one membership. A cache written for
// another host, cluster, scope or service type is ignored, since its ETag and
// configuration belong to a different layer of the configuration tree.
type cacheIdentity struct {
	Type     ServiceType `json:"type"`
	Scope    Scope       `json:"scope"`
	Cluster  string      `json:"cluster,omitempty"`
	Hostname string      `json:"hostname"`
}

// cacheFile is the on-disk format. It never holds the instance id.
type cacheFile struct {
	Identity cacheIdentity   `json:"identity"`
	ETag     string          `json:"etag"`
	Revision string          `json:"revision,omitempty"`
	Config   json.RawMessage `json:"config"`
}

func (c *Client) identity() cacheIdentity {
	return cacheIdentity{Type: c.svcType, Scope: c.Scope(), Cluster: c.cluster, Hostname: c.hostname}
}

// loadCache reads the cache file and adopts its ETag and configuration, so the
// first poll can answer 304. It reports whether a usable cache was found.
func (c *Client) loadCache() (Update, bool) {
	if c.cachePath == "" {
		return Update{}, false
	}
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			cclog.ComponentWarnf(component, "reading config cache %s: %v", c.cachePath, err)
		}
		return Update{}, false
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil || len(cf.Config) == 0 || !json.Valid(cf.Config) {
		cclog.ComponentWarnf(component, "ignoring malformed config cache %s", c.cachePath)
		return Update{}, false
	}
	if cf.Identity != c.identity() {
		cclog.ComponentInfof(component, "ignoring config cache %s written for another identity", c.cachePath)
		return Update{}, false
	}

	c.mu.Lock()
	c.etag, c.revision, c.blob = cf.ETag, cf.Revision, cf.Config
	c.mu.Unlock()
	return Update{Config: cf.Config, Revision: cf.Revision, Source: SourceCache}, true
}

// writeCache replaces the cache file atomically. The file may hold secrets
// and is created with mode 0600. Failures are logged, not returned: the cache
// is an optimisation for offline starts.
func (c *Client) writeCache(etag, revision string, blob json.RawMessage) {
	if c.cachePath == "" {
		return
	}
	data, err := json.Marshal(cacheFile{Identity: c.identity(), ETag: etag, Revision: revision, Config: blob})
	if err != nil {
		cclog.ComponentWarnf(component, "encoding config cache: %v", err)
		return
	}
	if err := writeFileAtomic(c.cachePath, data); err != nil {
		cclog.ComponentWarnf(component, "writing config cache %s: %v", c.cachePath, err)
	}
}

func (c *Client) removeCache() {
	if c.cachePath == "" {
		return
	}
	if err := os.Remove(c.cachePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		cclog.ComponentWarnf(component, "removing config cache %s: %v", c.cachePath, err)
	}
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
