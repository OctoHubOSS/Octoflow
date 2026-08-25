# Octoflow

Octoflow posts GitHub webhook events to Discord. Point a repository's (or organization's) webhook at Octoflow, tell it which channel to use, and it turns pushes, pull requests, issues, releases, and 30+ other GitHub event types into readable Discord embeds with per-repo routing and fine-grained control over which events actually get posted.

It's a rewrite of the original Git Logs project, kept running at the same `v2.gitlogs.xyz` domain for continuity with existing users.

---

## How it works

Octoflow is two services that share one Postgres database:

- **`bot`** (Rust, [poise](https://github.com/serenity-rs/poise)) the Discord-facing side. Slash commands to create webhooks, wire up repositories to channels, and configure event modifiers.
- **`webserver`** (Go) the GitHub-facing side. Receives webhook deliveries, verifies them against the webhook's secret, checks them against any event modifiers, and posts the resulting embed to Discord.

A webhook, once created, gets its own unique ID and secret. Add that as a webhook in GitHub's repository (or organization) settings, and set the `Secret` field and Content-Type header as instructed when the webhook is created. Each **repository** you register against that webhook maps to one Discord channel.

---

## Commands

All commands are slash commands and require the **Manage Server** permission (except `/help`, which anyone can use).

| Command | What it does |
|---|---|
| `/newhook` | Creates a new webhook in this guild (max 5 per guild). The webhook ID and secret are DMed to you — never posted in a channel. |
| `/edithook` | Updates a webhook's comment, broken status, or secret. |
| `/resetsecret` | Rotates a webhook's secret and DMs you the new one. |
| `/delhook` | Deletes a webhook and everything registered under it. |
| `/list` | Lists every webhook in the guild, each with its registered repos and their channels. |
| `/newrepo` | Registers a GitHub repository against a webhook, routed to a channel. |
| `/editrepo` | Updates a repo's owner/name or its destination channel. |
| `/setrepochannel` | Updates just a repo's destination channel. |
| `/delrepo` | Removes a registered repository. |
| `/eventmod create` | Creates an event modifier: a blacklist, a whitelist, or a per-repo redirect to a different channel, matched against event names (wildcards supported: `*`, `?`). |
| `/eventmod edit` | Updates an existing event modifier. Only the fields you pass are changed. |
| `/eventmod list` | Lists every event modifier in the guild, or just the ones on one webhook. |
| `/eventmod delete` | Deletes an event modifier. |
| `/backup` | Exports a webhook's repos and event modifiers as a downloadable file. |
| `/restore` | Re-imports a backup file into a webhook, inserting or updating as needed. |
| `/help`, `/simplehelp` | Command reference. |

### Event modifiers

A webhook posts every event, for every registered repo, to that repo's channel by default. Event modifiers change that per repo:

- **Blacklist** — matching events are dropped.
- **Whitelist** — only matching events are posted; anything else is dropped.
- **Redirect** — matching events go to a different channel instead of the repo's default.

Modifiers are evaluated in priority order (highest first); the first blacklist or unmatched whitelist that applies wins. Event names are matched case-insensitively, and `*`/`?` wildcards are supported (e.g. `pull_request*` matches both `pull_request` and `pull_request_review`).

---

## Supported events

Octoflow currently posts embeds for these GitHub webhook event types:

`branch_protection_rule`, `check_run`, `check_suite`, `commit_comment`, `create`, `delete`, `dependabot_alert`, `deployment`, `deployment_status`, `discussion`, `discussion_comment`, `fork`, `gollum`, `issue_comment`, `issues`, `label`, `member`, `membership`, `milestone`, `page_build`, `ping`, `public`, `pull_request`, `pull_request_review`, `pull_request_review_comment`, `push`, `release`, `repository`, `star`, `status`, `team`, `team_add`, `watch`, `workflow_dispatch`, `workflow_job`, `workflow_run`

For the full, current list with descriptions, see the live [events list view](https://v2.gitlogs.xyz/api/events/listview). For what each event's payload actually contains, see [GitHub's webhook events documentation](https://docs.github.com/en/webhooks/webhook-events-and-payloads).

Events GitHub sends that aren't in the list above are accepted (no error to GitHub) but silently ignored — no Discord message is sent for them.

---

## Self-hosting

### Requirements

- PostgreSQL
- Rust (stable) for the bot
- Go 1.25+ for the webserver

### Database

```sql
CREATE DATABASE github;
\c github
\i schema.sql
```

### Configuration

Both services read a `config.yaml` in their working directory (each writes a `config.yaml.sample` on first run showing the exact shape expected).

**`bot/config.yaml`:**

```yaml
database_url: postgres://user:pass@localhost/github
token: your-discord-bot-token
api_url:
  - https://your-webserver-domain
proxy_url: http://127.0.0.1:3219 # optional
```

**`webserver/config.yaml`:**

```yaml
token: your-discord-bot-token       # same bot token as above
postgres_url: postgresql://user:pass@localhost/github
port: ":19318"
api_url: https://your-webserver-domain
```

`bot/.env` also needs a bare `DATABASE_URL` — this is only used by `sqlx`'s compile-time query checking, not read at runtime.

### Compiling

- `cd bot && make selfhost`
- `cd webserver && make`

### Running

Two long-running processes, ideally as separate systemd services:

- `bot`: `make run` (from the `bot` folder)
- `webserver`: `./webserver` (from the `webserver` folder)

A `206` status code from the webhook endpoint means the delivery was accepted but the repository sending it isn't registered against that webhook — check `/newrepo`.

---

## License

AGPL-3.0. See [LICENSE](LICENSE).
