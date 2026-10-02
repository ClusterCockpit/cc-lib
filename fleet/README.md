<!--
---
title: Fleet client
description: Register with the cc-backend fleet service, receive central configuration and discover peers
categories: [cc-lib]
tags: ['Admin', 'Developer']
weight: 2
hugo_path: docs/reference/cc-lib/fleet/_index.md
---
-->

# Fleet client

Package `fleet` is the client for the cc-backend fleet service. With it a
ClusterCockpit service (cc-metric-store, cc-metric-collector, cc-slurm-adapter,
…) does the following:

- registers with cc-backend and gets an instance id,
- receives its configuration from the central configuration tree, with
  ETag-based polling,
- stays marked alive by sending heartbeats over NATS, or over REST when NATS
  is unavailable,
- receives discovery rosters of the peers it depends on,
- deregisters on shutdown.

The package also holds the wire definitions shared by client and server: the
service type codes, the registration bodies, the roster entry, and the
line-protocol encoding of heartbeats and rosters.

For the server side (enabling the service, authoring the configuration tree)
see `internal/fleet/README.md` in cc-backend.

## Configuration

The application keeps a local `fleet` section. It only says how to reach
cc-backend and never comes from the fleet service itself.

```json
{
  "fleet": {
    "url": "https://cc.example.org",
    "token": "<jwt with api role>",
    "cluster": "fritz",
    "heartbeat-interval": "30s",
    "config-poll-interval": "60s",
    "heartbeat-subject": "cc.fleet.event",
    "cache-path": "/var/lib/cc-metric-collector/fleet-config.json"
  }
}
```

| Key                        | Default              | Meaning                                                   |
| -------------------------- | -------------------- | --------------------------------------------------------- |
| `url`                      | — (required)         | cc-backend base URL                                       |
| `token`                    | —                    | JWT with the `api` role                                   |
| `cluster`                  | — (infra scope)      | Cluster of this service; empty registers in infra scope   |
| `hostname`                 | system hostname      | Hostname to register with                                 |
| `heartbeat-interval`       | `30s`                | Keep it well below cc-backend's `stale-after`             |
| `config-poll-interval`     | `60s`                | The unchanged case is a header-only round trip            |
| `heartbeat-subject`        | — (REST heartbeats)  | NATS heartbeat subject, as configured in cc-backend       |
| `discovery-subject-prefix` | `cc.fleet.discovery` | Prefix of the roster subjects                             |
| `cache-path`               | — (no cache)         | Last configuration, for starting while cc-backend is down |
| `timeout`                  | `10s`                | Timeout of a single HTTP request                          |

`CC_FLEET_TOKEN`, or the file named by `CC_FLEET_TOKEN_FILE`, overrides
`token`. `ConfigSchema` holds the JSON schema of the section.

## Usage

```go
cfg, err := fleet.ParseConfig(ccconfig.GetPackageConfig("fleet"))
client, err := fleet.New(fleet.Options{
    Config:      cfg,
    ServiceType: fleet.ServiceMetricCollector,
    Meta:        map[string]string{"version": version}, // broadcast: no secrets
    NATS:        natsClient.Load, // func() *nats.Client, nil until connected
})

initial, err := client.Bootstrap(ctx, 10*time.Second)
// err == nil:             initial.Source is SourceFleet, or SourceNone (nothing authored)
// errors.Is(err, fleet.ErrUnreachable): initial is the cached config (SourceCache) or none
// errors.Is(err, fleet.ErrUnauthorized): token or IP allowlist problem
applyConfig(initial.Config) // nil: run with the local configuration

go client.Run(ctx)
defer client.Close(shutdownCtx) // stops Run, then deregisters

for {
    select {
    case u := <-client.Configs():
        applyConfig(u.Config) // only sent when the configuration really changed
    case providers := <-client.Rosters():
        replaceProviders(providers) // full list, not a delta
    case <-ctx.Done():
        return
    }
}
```

`Configs()` and `Rosters()` each hold at most one value. An update that has
not been read yet is replaced by the newer one, so a slow main loop always
sees the latest state and never blocks the client.

Merging the received configuration with the local one is left to the
application. Which keys must stay local (the fleet section itself, paths,
credentials) and which need a restart differs from service to service.

## Behaviour

- **Heartbeats** go to `heartbeat-subject` while the NATS client is connected,
  and over REST otherwise. A disconnected NATS client never buffers them
  silently.
- **Unknown instance.** When cc-backend answers `404` (the id was rotated or
  deregistered), `Run` registers again and pulls the configuration at once.
  NATS heartbeats cannot detect this, so the config poll does.
- **Failures** are logged once as a warning, repeated at debug level and
  followed by an info message on recovery. Registration attempts back off
  exponentially (1 s up to 5 min), so a rejected token does not hammer
  cc-backend.
- **Change detection** compares the configuration body. A cosmetic edit on
  the server, or a response without an ETag, does not deliver the same
  configuration again.
- **Discovery.** `Run` subscribes to
  `<prefix>.<cluster|infra>.<service type>` as soon as the NATS provider
  returns a connected client, and again if it starts returning a different
  client.
- **Cache.** The cache file is written atomically with mode 0600, because the
  configuration may contain secrets. It never contains the instance id and is
  ignored if it was written for another service type, scope, cluster or host.
  A `204` from cc-backend deletes it.

## Low-level API

`Register`, `PullConfig`, `Heartbeat` and `Deregister` are exported for
applications that drive their own loop. Errors unwrap to `ErrUnknownInstance`
(404), `ErrUnauthorized` (401/403) or `ErrNotRegistered`. Other unexpected
statuses are `*StatusError`.

`EncodeHeartbeat`, `ParseHeartbeat`, `DecodeHeartbeats`, `EncodeRoster`,
`DecodeRoster`, `DiscoverySubject` and `Bucket` implement the NATS wire format
for both sides.
