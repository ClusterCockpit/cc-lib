// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package fleet is the client side of the cc-backend fleet service. With it a
// ClusterCockpit service registers with cc-backend, receives its configuration
// from the central configuration tree, keeps itself marked as alive and learns
// about the peers it depends on.
//
// The package also holds the wire definitions shared by client and server:
// the service type codes, the registration request and response bodies, the
// discovery roster entry, and the line-protocol encoding of heartbeats and
// rosters. Both sides should use them instead of keeping their own copies.
//
// # Lifecycle
//
// A member walks through these steps:
//
//  1. Register over authenticated REST and receive an instance id.
//  2. Pull the merged configuration. The ETag makes later polls a header-only
//     round trip, and 204 means "no configuration authored", not an error.
//  3. Send a heartbeat periodically, over NATS when a heartbeat subject is
//     configured and the connection is up, and over REST otherwise.
//  4. Optionally, receive discovery rosters on
//     <prefix>.<cluster|infra>.<service type>. Each roster replaces the
//     previous list in full; it is not a delta.
//  5. Deregister on shutdown.
//
// [Client.Bootstrap] covers steps 1 and 2 at startup and falls back to the
// optional on-disk cache when cc-backend cannot be reached. [Client.Run]
// drives steps 2 to 4, re-registers when cc-backend reports the instance as
// unknown, and delivers changes on [Client.Configs] and [Client.Rosters].
// [Client.Close] stops Run and performs step 5.
//
// # Security
//
// The instance id is a bearer credential for heartbeat, config pull and
// deregistration. The client never logs it and never writes it to the cache
// file. Registration metadata ([Options.Meta]) is broadcast unauthenticated in
// discovery rosters, so it must carry connection hints only, never secrets.
// The cached configuration may contain secrets, so the cache file is written
// with mode 0600.
package fleet
