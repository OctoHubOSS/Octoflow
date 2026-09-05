# Changelog

All notable changes to Octoflow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/). History
before v2.0.0 wasn't tracked in this format, so this file starts there and
moves forward.

## [Unreleased]

## [2.3.0] - 2026-09-05

### Added
- `octoflow.ca/commands`: a searchable, at-a-glance commands page (separate
  from the full `/docs/commands` reference), linked in the site nav.
- Omniplex bot-list stats reporting: posts server/shard/user counts and a
  self-reported `online` presence to `POST /bots/stats` on
  `api.omniplex.gg` every 5 minutes. Opt-in via a new `omniplex_token` config
  field - left unset, the task logs once and never posts.
- Omniplex command list sync: on every Ready, `PUT /bots/{id}/commands`
  replaces the bot's documented command list on Omniplex with the live
  set (name, description, usage, category), flattened from the same
  command tree `/help` reads, skipping anything marked hidden. Shares the
  `omniplex_token` config field with stats reporting.
- Omniplex changelog sync: on every Ready, parses the latest released
  entry straight out of this file (embedded into the binary at compile
  time) and posts it via `POST /bots/{id}/changelogs`, first checking
  `GET /bots/{id}/changelogs` so the same version is never posted twice
  across restarts between releases.
- Autocomplete for `webhook_id`/`repo_id`/`modifier_id` parameters across
  every command that takes one, instead of requiring copy-paste from
  `/list`. Scoped to the current guild, capped at Discord's own 25-choice
  limit, each choice labeled with its comment/repo name/event summary
  alongside the ID.
- Dead-webhook detection: a webhook with at least one repo linked but no
  event in 14+ days gets automatically marked broken (same flag `/edithook
  broken:true` already sets) and a one-time DM via the webserver's own
  Discord REST client explaining why, pointing at `/list` and
  `/resetsecret` to fix it and `/edithook broken:false` to re-enable it.
  Fully reversible, and marking it broken is what stops the checker from
  ever re-nagging about the same webhook. New `webhooks.last_nudged_at`
  column.
- `/testevent`: preview an event's rendered embed and see which channel(s)
  it would route to, without waiting for a real GitHub delivery. Covers
  `push`, `pull_request`, `issues`, `issue_comment`, `release`, `star`,
  `fork`, and `ping` with schema-accurate sample payloads, run through the
  exact same renderers and event-modifier evaluation real events use.
  Always a private/ephemeral preview - nothing is ever actually sent to the
  resolved channel. Backed by a new webserver endpoint,
  `POST /api/dashboard/webhooks/{id}/simulate`.


## [2.2.0] - 2026-08-26

### Added
- Thread-per-PR/issue mode (`use_threads` on a repo): opens a Discord thread
  when an issue or PR is opened, then routes its follow-up activity
  (comments, reviews, status changes) into that thread instead of posting
  flat messages into the channel. Backed by a new `issue_threads` table.
  Falls back to a normal channel post if the bot only sees an issue/PR after
  it was already opened.
- Burst batching (`batch_events` on a webhook): collapses rapid consecutive
  `push` events into one combined summary embed instead of one per push,
  using a 20-second buffering window. Only `push` is batched for now.
- Dashboard analytics: an events-per-day chart and a top-event-types
  breakdown on each guild's dashboard page, backed by a new `event_metrics`
  table and a `/api/dashboard/guilds/{guildId}/analytics` endpoint.
- `/edithook` gained a `batch_events` toggle, `/editrepo` gained a
  `use_threads` toggle - both optional, both mirrored as checkboxes in the
  dashboard's webhook/repo edit dialogs.
- Bot admin panel (`octoflow.ca/admin`): global stats (guild/webhook/repo
  counts, event volume, bot heartbeat), guild ban/unban, a webhook-log
  search tool, and an admin action audit trail. Gated by a Discord user ID
  allowlist (`admin_user_ids` in the webserver config), checked on every
  request rather than just at login.
- `ROADMAP.md` - a running, non-committal list of feature ideas under
  consideration for future releases.
- `status-checker/`: a standalone Go service replacing the GitHub
  Actions-based external status check. Runs as its own long-lived process
  (deployable to any VM or Dokploy via its included `Dockerfile`), checks
  `/api/health` on its own schedule, and serves the same NDJSON history
  format over plain HTTP instead of committing it to git - built because
  GitHub's scheduled workflows proved unreliable (silently delayed or
  dropped runs). The docs site now reads from `EXTERNAL_STATUS_HISTORY_URL`
  when set, falling back to the old GitHub raw URL otherwise, so cutover
  needs no code changes.
- CI: `webserver.yml` and `status-checker.yml` GitHub Actions workflows,
  building and testing the Go webserver and the new status-checker service
  the same way `rust.yml` already does for the bot.
- `/api/stats/summary`: a new public, unauthenticated endpoint returning
  aggregate totals (webhooks configured, repos connected, events processed
  in the last 24h/7d/30d/all-time, plus the existing guild/member/shard
  counts) with no guild-identifying data. Powers an expanded
  `octoflow.ca/stats` page.

### Changed
- `/help`'s quick-links are now the last page of the command pagination
  instead of a second message sent alongside it - everything lives in one
  embed, reachable via Next/Previous or the page-select menu.
- The status page's uptime strip no longer risks pushing the page into
  horizontal scroll on narrow viewports - its per-day tooltip is now clipped
  locally instead of relying solely on a page-wide backstop.

## [2.1.0] - 2026-08-26

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
- Changelog page (`octoflow.ca/changelog`), rendered straight from this file.
- `/api/health` and `/api/status/history` endpoints, backed by a new
  `bot_heartbeat` table (updated by the bot every minute) and a
  `status_snapshots` table populated by a background health-check job.
- External status checker: a GitHub Actions job hits `/api/health` from
  outside our own infrastructure every 5 minutes and records the result to
  `status-history.ndjson`. It fills in gaps on the status page for the one
  case the in-process checker structurally can't record: the webserver or
  its database being fully down (nothing left running to write the outage
  down).
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
- `/help`'s paginated command list is now built in-house instead of
  depending on the third-party `botox` crate. `/simplehelp` is unaffected -
  it already used poise's own built-in help renderer.

### Fixed
- Documentation and troubleshooting guidance corrected to match current bot
  behavior (event modifier priority ordering, the real 206-status cause,
  DM-failure rollback behavior on `/newhook` and `/resetsecret`).
- A long tooltip on the status page's uptime history could force the whole
  page to scroll horizontally on mobile.

## [2.0.0] - 2026-08-25

Baseline for this changelog. Reflects the bot as it stood before the changes
above: slash-command-only (no more `git!` prefix), full CRUD for webhooks,
repos, and event modifiers, backups/restore, 35 supported GitHub event types
with wildcard (`*`/`?`) matching in event modifiers, and the webserver's
webhook diagnostic page, per-event audit logs, and public event-list API.
