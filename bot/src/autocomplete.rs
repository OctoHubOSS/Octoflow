//  Copyright (C) 2026 NodeByte LTD

use poise::serenity_prelude as serenity;
use sqlx::Row;

use crate::Context;

fn truncate_chars(s: &str, max: usize) -> String {
    s.chars().take(max).collect()
}

pub async fn webhook_id(ctx: Context<'_>, partial: &str) -> Vec<serenity::AutocompleteChoice<'static>> {
    let Some(guild_id) = ctx.guild_id() else { return Vec::new() };
    let partial_lower = partial.to_lowercase();

    let rows = sqlx::query("SELECT id, comment FROM webhooks WHERE guild_id = $1 ORDER BY created_at DESC")
        .bind(guild_id.to_string())
        .fetch_all(&ctx.data().pool)
        .await
        .unwrap_or_default();

    rows.into_iter()
        .filter_map(|row| {
            let id: String = row.try_get("id").ok()?;
            let comment: String = row.try_get("comment").ok()?;
            Some((id, comment))
        })
        .filter(|(id, comment)| id.to_lowercase().contains(&partial_lower) || comment.to_lowercase().contains(&partial_lower))
        .take(25)
        .map(|(id, comment)| {
            let label = truncate_chars(&format!("{} — {}", comment, id), 100);
            serenity::AutocompleteChoice::new(label, id)
        })
        .collect()
}

pub async fn repo_id(ctx: Context<'_>, partial: &str) -> Vec<serenity::AutocompleteChoice<'static>> {
    let Some(guild_id) = ctx.guild_id() else { return Vec::new() };
    let partial_lower = partial.to_lowercase();

    let rows = sqlx::query("SELECT id, repo_name FROM repos WHERE guild_id = $1 ORDER BY created_at DESC")
        .bind(guild_id.to_string())
        .fetch_all(&ctx.data().pool)
        .await
        .unwrap_or_default();

    rows.into_iter()
        .filter_map(|row| {
            let id: String = row.try_get("id").ok()?;
            let repo_name: String = row.try_get("repo_name").ok()?;
            Some((id, repo_name))
        })
        .filter(|(id, repo_name)| id.to_lowercase().contains(&partial_lower) || repo_name.to_lowercase().contains(&partial_lower))
        .take(25)
        .map(|(id, repo_name)| {
            let label = truncate_chars(&format!("{} — {}", repo_name, id), 100);
            serenity::AutocompleteChoice::new(label, id)
        })
        .collect()
}

pub async fn modifier_id(ctx: Context<'_>, partial: &str) -> Vec<serenity::AutocompleteChoice<'static>> {
    let Some(guild_id) = ctx.guild_id() else { return Vec::new() };
    let partial_lower = partial.to_lowercase();

    let rows = sqlx::query(
        "SELECT id, webhook_id, events, blacklisted, whitelisted FROM event_modifiers WHERE guild_id = $1 ORDER BY priority DESC",
    )
    .bind(guild_id.to_string())
    .fetch_all(&ctx.data().pool)
    .await
    .unwrap_or_default();

    rows.into_iter()
        .filter_map(|row| {
            let id: String = row.try_get("id").ok()?;
            let webhook_id: String = row.try_get("webhook_id").ok()?;
            let events: Vec<String> = row.try_get("events").ok()?;
            let blacklisted: bool = row.try_get("blacklisted").ok()?;
            let whitelisted: bool = row.try_get("whitelisted").ok()?;
            Some((id, webhook_id, events, blacklisted, whitelisted))
        })
        .filter(|(id, ..)| id.to_lowercase().contains(&partial_lower))
        .take(25)
        .map(|(id, webhook_id, events, blacklisted, whitelisted)| {
            let kind = if whitelisted {
                "Whitelist"
            } else if blacklisted {
                "Blacklist"
            } else {
                "No-op"
            };
            let label = truncate_chars(
                &format!("{} {} on {} — {}", kind, events.join(","), webhook_id, id),
                100,
            );
            serenity::AutocompleteChoice::new(label, id)
        })
        .collect()
}
