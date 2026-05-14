CREATE TABLE guilds (
    id TEXT PRIMARY KEY NOT NULL,
    banned BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE webhooks (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    comment TEXT NOT NULL, -- A comment to help identify the webhook
    broken BOOLEAN NOT NULL DEFAULT FALSE,
    secret TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'github', -- 'github' or 'gitlab'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by TEXT NOT NULL
);

CREATE TABLE repos (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE,
    repo_name TEXT NOT NULL,
    channel_id TEXT NOT NULL, -- Channel ID to post to
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by TEXT NOT NULL
);

CREATE TABLE event_modifiers (
    id TEXT PRIMARY KEY NOT NULL,
    guild_id TEXT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE ON UPDATE CASCADE,
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE ON UPDATE CASCADE, -- Webhook to apply to
    repo_id TEXT REFERENCES repos(id) ON DELETE CASCADE ON UPDATE CASCADE, -- Optional, if not set, will assume all repos
    events TEXT[] NOT NULL DEFAULT '{}', -- Events to capture in this modifier
    blacklisted boolean not null default false, -- Whether or not these events are blacklisted or not
    whitelisted boolean not null default false, -- Whether or not only these events can be sent
    redirect_channel TEXT, -- Channel ID to redirect to, otherwise use default channel
    priority INTEGER NOT NULL, -- Priority to apply the modifiers in, applied in descending order
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
