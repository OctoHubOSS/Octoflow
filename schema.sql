CREATE TABLE guilds (
    id TEXT PRIMARY KEY NOT NULL,
    banned BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE webhooks (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    comment TEXT NOT NULL,
    broken BOOLEAN NOT NULL DEFAULT FALSE,
    secret TEXT NOT NULL,
    batch_events BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by TEXT NOT NULL,
    last_nudged_at TIMESTAMPTZ
);

CREATE TABLE repos (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    repo_name TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    use_threads BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by TEXT NOT NULL
);

CREATE TABLE issue_threads (
    id BIGSERIAL PRIMARY KEY,
    repo_id TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE ON UPDATE CASCADE,
    issue_number INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('issue', 'pull_request')),
    thread_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (repo_id, issue_number, kind)
);

CREATE TABLE event_modifiers (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    repo_id TEXT REFERENCES repos(id) ON DELETE CASCADE ON UPDATE CASCADE,
    events TEXT[] NOT NULL DEFAULT '{}',
    blacklisted boolean not null default false,
    whitelisted boolean not null default false,
    redirect_channel TEXT,
    priority INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by TEXT NOT NULL
);

create table webhook_logs (
    log_id text primary key not null,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    webhook_id text not null references webhooks (id) ON UPDATE CASCADE ON DELETE CASCADE,
    entries text[] not null default '{}'
);

CREATE TABLE bot_heartbeat (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    guild_count INTEGER NOT NULL,
    member_count BIGINT NOT NULL,
    shard_count INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE status_snapshots (
    id BIGSERIAL PRIMARY KEY,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    database_up BOOLEAN NOT NULL,
    discord_up BOOLEAN NOT NULL,
    db_latency_ms INTEGER NOT NULL
);

CREATE TABLE event_metrics (
    id BIGSERIAL PRIMARY KEY,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    repo_id TEXT REFERENCES repos(id) ON DELETE CASCADE ON UPDATE CASCADE,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Append-only trail of what an admin did in the bot admin panel (banned a
-- guild, etc.) - separate from webhook_logs, which is per-webhook delivery
-- debug output, not an accountability record.
CREATE TABLE admin_audit_log (
    id BIGSERIAL PRIMARY KEY,
    admin_user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT,
    detail TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
