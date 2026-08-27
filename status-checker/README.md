# status-checker

A standalone, long-running replacement for the old GitHub Actions-based
external status check (`.github/workflows/external-status-check.yml` +
`scripts/check-status.sh`). Same purpose - hit Octoflow's `/api/health` from
outside its own infrastructure so an outage still gets recorded even if the
webserver or its database is fully down - just running as a real service
instead of a GitHub cron, since GitHub's scheduled workflows have proven
unreliable (silently delayed or dropped runs).

It's a separate Go module from `webserver/` on purpose: it has no
dependency on Postgres, Discord, or anything else Octoflow-specific, and
deploying it somewhere entirely separate from the main stack is the point.

## Running it

```
go run .
```

Or build and run the binary directly. Config is all environment variables,
all optional:

| Variable | Default | Meaning |
|---|---|---|
| `TARGET_URL` | `https://v2.gitlogs.xyz` | Base URL to check; `/api/health` is appended |
| `DATA_FILE` | `/data/status-history.ndjson` | Where checks are recorded |
| `LISTEN_ADDR` | `:8090` | Address the HTTP server binds to |
| `CHECK_INTERVAL` | `5m` | How often to check (Go duration syntax) |
| `CHECK_TIMEOUT` | `10s` | Per-check HTTP timeout |
| `MAX_LINES` | `30000` | Oldest records are trimmed past this many lines |

It checks once immediately on startup, then on the interval.

## Endpoints

- `GET /status-history.ndjson` - the raw recorded history, one JSON object
  per line, oldest first. Same shape `scripts/check-status.sh` used to
  produce, so anything already parsing that format needs no changes beyond
  the URL.
- `GET /healthz` - plain 200 OK, for Dokploy/Docker's own health check of
  this service (not of Octoflow).

## Deploying with Dokploy

1. Point a Dokploy app at this directory (it'll find the `Dockerfile`), or
   use the included `docker-compose.yml` if you'd rather deploy via compose.
2. Mount a persistent volume at `/data` - it's already declared as a
   `VOLUME` in the Dockerfile, so Dokploy should offer to attach storage
   there. Without it, history is lost on every redeploy/restart.
3. Set `TARGET_URL` if you're not pointing this at the default production
   API.
4. Once it's up, note the public URL Dokploy gives it (or whatever domain
   you point at it).

## Wiring it into the docs site

Set `EXTERNAL_STATUS_HISTORY_URL` in the `octoflow-docs` deployment's
environment to `https://<wherever-this-is-deployed>/status-history.ndjson`.
`lib/api.ts` falls back to the old GitHub raw URL if that env var isn't
set, so this is a zero-downtime cutover - set it whenever the new service
has a few data points recorded, no code changes needed.

Once you've confirmed this is reliably recording, the old
`.github/workflows/external-status-check.yml` can be disabled or deleted -
it's redundant with this running, not required alongside it.
