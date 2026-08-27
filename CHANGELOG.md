# Changelog

All notable changes to Octoflow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/). History
before v2.0.0 wasn't tracked in this format, so this file starts there and
moves forward.

## [Unreleased]

### Added
- Dashboard: Discord OAuth-gated web UI (`octoflow.ca/dashboard`) for viewing
  and managing webhooks, linked repos, and event modifiers without needing
  Discord slash commands.
- `/api/dashboard/*` backend: guild/webhook/repo/modifier read endpoints plus
  full create/edit/delete, a channel list for picker dropdowns, and a
  dedicated secret-reset endpoint. Enforces the same limits as the slash
  commands (5 webhooks/guild, 10 modifiers/webhook).
- Live status page (`octoflow.ca/status`): current API/bot health plus 90-day
  uptime history. Replaces the previous external status page.
- Live stats page (`octoflow.ca/stats`): server/user/shard counts pulled
  directly from the bot.
- `/api/health` and `/api/status/history` endpoints, backed by a new
  `bot_heartbeat` table (updated by the bot every minute) and a
  `status_snapshots` table populated by a background health-check job.
- Quick-links section (docs, status, dashboard, support server) appended to
  `/help` and `/simplehelp`.
- Terms of Service and Privacy Policy pages.

### Changed
- `/api/counts` now reads live guild/member/shard counts from the bot's
  heartbeat table. It previously always returned `0,0,0`, since it read from
  a Discord gateway cache that the webserver process never actually
  populates (it doesn't open its own gateway connection).
- `/newhook` and `/resetsecret` DM messages are now structured embeds with
  numbered setup steps and a docs link, replacing the previous plain-text
  message.
- `/list`, `/eventmod list`, `/eventmod create`, and `/restore` use a
  consistent embed style (brand color, docs-link footer) and clearer field
  labels.
- Error and confirmation messages across core commands now say what to do
  next (which command to run) instead of just stating the problem.

### Fixed
- Documentation and troubleshooting guidance corrected to match current bot
  behavior (event modifier priority ordering, the real 206-status cause,
  DM-failure rollback behavior on `/newhook` and `/resetsecret`).

## [2.0.0] - 2026-08-25

Baseline for this changelog. Reflects the bot as it stood before the changes
above: slash-command-only (no more `git!` prefix), full CRUD for webhooks,
repos, and event modifiers, backups/restore, 35 supported GitHub event types
with wildcard (`*`/`?`) matching in event modifiers, and the webserver's
webhook diagnostic page, per-event audit logs, and public event-list API.
