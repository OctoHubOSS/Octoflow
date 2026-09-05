//  Copyright (C) 2026 NodeByte LTD

use poise::serenity_prelude::ChannelId;
use poise::CreateReply;
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error, autocomplete, embeds};

const NO_WEBHOOKS_MSG: &str = "You don't have any webhooks in this guild. Run `/newhook` to create one.";

#[poise::command(
    category = "Event Modifiers",
    slash_command,
    guild_cooldown = 10,
    subcommands("create", "delete", "list", "edit")
)]
pub async fn eventmod(_ctx: Context<'_>) -> Result<(), Error> {
    Ok(())
}

#[poise::command(
    slash_command,
    guild_only,
    guild_cooldown = 60,
    required_permissions = "MANAGE_GUILD"
)]
#[allow(clippy::too_many_arguments)]
pub async fn create(
    ctx: Context<'_>,
    #[description = "The webhook ID"]
    #[autocomplete = "autocomplete::webhook_id"]
    webhook_id: String,
    #[description = "The events to match against, comma/space seperated"] events: String,
    #[description = "Blacklist the events"] blacklisted: bool,
    #[description = "Whitelist the events. Other events will not be allowed"] whitelisted: bool,
    #[description = "Priority. Use 0 for normal priority"] priority: Option<i32>,
    #[description = "Repository ID, will match all if unset"]
    #[lazy]
    #[autocomplete = "autocomplete::repo_id"]
    repo_id: Option<String>,
    #[description = "Redirect channel ID"] redirect_channel: Option<ChannelId>,
) -> Result<(), Error> {
    let data = ctx.data();
    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        return Err(NO_WEBHOOKS_MSG.into());
    }

    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    let count = webhook_count.count.unwrap_or_default();

    if count == 0 {
        Err(NO_WEBHOOKS_MSG.into())
    } else {
        let webhook = sqlx::query!(
            "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
            webhook_id,
            ctx.guild_id().unwrap().to_string()
        )
        .fetch_one(&data.pool)
        .await?;

        if webhook.count.unwrap_or_default() == 0 {
            return Err("That webhook doesn't exist. Use `/newhook` to create one, or `/list` to see your existing webhooks.".into());
        }

        let mut parsed_repo_id = repo_id.clone();

        if let Some(ref inner_repo_id) = repo_id {
            if inner_repo_id.is_empty() || inner_repo_id == "None" || inner_repo_id == "none" {
                parsed_repo_id = None;
            } else {

                let repo = sqlx::query!(
                    "SELECT COUNT(1) FROM repos WHERE id = $1 AND webhook_id = $2",
                    inner_repo_id,
                    webhook_id
                )
                .fetch_one(&data.pool)
                .await?;

                if repo.count.unwrap_or_default() == 0 {
                    return Err("That repo doesn't exist on this webhook. Use `/newrepo` to link it first.".into());
                }
            }
        }

        let events = events
            .replace('`', "")
            .replace(',', " ")
            .replace("  ", " ")
            .to_lowercase()
            .split(' ')
            .map(|s| s.to_string())
            .collect::<Vec<String>>();

        let modifier_count = sqlx::query!(
            "SELECT COUNT(1) FROM event_modifiers WHERE webhook_id = $1",
            webhook_id
        )
        .fetch_one(&data.pool)
        .await?;

        let count = modifier_count.count.unwrap_or_default();

        if count >= 10 {
            return Err("You can only have 10 event modifiers per webhook. Delete one with `/eventmod delete` first.".into());
        }

        let modifier_id = Alphanumeric.sample_string(&mut rand::thread_rng(), 32);
        sqlx::query!(
            "INSERT INTO event_modifiers (id, webhook_id, events, repo_id, blacklisted, whitelisted, redirect_channel, guild_id, priority, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
            modifier_id,
            webhook_id,
            &events,
            parsed_repo_id,
            blacklisted,
            whitelisted,
            redirect_channel.map(|c| c.to_string()),
            ctx.guild_id().unwrap().to_string(),
            priority.unwrap_or_default(),
            ctx.author().id.to_string(),
            ctx.author().id.to_string(),
        )
        .execute(&data.pool)
        .await?;

        let kind = if whitelisted {
            "Whitelist"
        } else if blacklisted {
            "Blacklist"
        } else {
            "No-op (both blacklisted and whitelisted are false)"
        };

        ctx.send(CreateReply::default().embed(
            embeds::base()
                .title("Event modifier created")
                .url(format!("{}/commands/modifiers", embeds::DOCS_URL))
                .field("ID", format!("`{}`", modifier_id), true)
                .field("Type", kind, true)
                .field("Priority", priority.unwrap_or_default().to_string(), true)
                .field("Events", format!("`{}`", events.join(", ")), false)
                .field(
                    "Repo scope",
                    parsed_repo_id.unwrap_or_else(|| "All repos on this webhook".to_string()),
                    true,
                )
                .field(
                    "Redirect channel",
                    redirect_channel.map(|c| format!("<#{}>", c)).unwrap_or_else(|| "None".to_string()),
                    true,
                ),
        )).await?;

        Ok(())
    }
}

#[poise::command(
    slash_command,
    guild_only,
    guild_cooldown = 60,
    required_permissions = "MANAGE_GUILD"
)]
pub async fn delete(
    ctx: Context<'_>,
    #[description = "The modifier ID"]
    #[autocomplete = "autocomplete::modifier_id"]
    modifier_id: String,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        return Err(NO_WEBHOOKS_MSG.into());
    }

    let modifier_count = sqlx::query!(
        "SELECT COUNT(1) FROM event_modifiers WHERE guild_id = $1 AND id = $2",
        ctx.guild_id().unwrap().to_string(),
        modifier_id
    )
    .fetch_one(&data.pool)
    .await?;

    let count = modifier_count.count.unwrap_or_default();

    if count == 0 {
        return Err("That modifier doesn't exist. Use `/eventmod list` to see your existing modifiers.".into());
    }

    sqlx::query!("DELETE FROM event_modifiers WHERE id = $1", modifier_id)
        .execute(&data.pool)
        .await?;

    ctx.say("Event modifier deleted.").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn list(
    ctx: Context<'_>,
    #[description = "Filter to a specific webhook ID"]
    #[autocomplete = "autocomplete::webhook_id"]
    webhook_id: Option<String>,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild_id = ctx.guild_id().unwrap().to_string();

    let modifiers = sqlx::query!(
        "SELECT id, webhook_id, repo_id, events, blacklisted, whitelisted, redirect_channel, priority, created_at
         FROM event_modifiers
         WHERE guild_id = $1 AND ($2::text IS NULL OR webhook_id = $2)
         ORDER BY priority DESC",
        guild_id,
        webhook_id
    )
    .fetch_all(&data.pool)
    .await?;

    if modifiers.is_empty() {
        ctx.say(format!("No event modifiers found. Use `/eventmod create` to create one. See the [event modifiers guide]({}/commands/modifiers) for wildcard syntax and examples.", embeds::DOCS_URL)).await?;
        return Ok(());
    }

    let mut cr = CreateReply::default().content("Here are the event modifiers in this guild, in evaluation order (highest priority first):");

    for modifier in modifiers {
        let kind = if modifier.whitelisted {
            "Whitelist"
        } else if modifier.blacklisted {
            "Blacklist"
        } else {
            "No-op (has no effect)"
        };

        cr = cr.embed(
            embeds::base()
                .title(format!("{} Modifier", kind))
                .field("ID", format!("`{}`", modifier.id), false)
                .field("Webhook", format!("`{}`", modifier.webhook_id), true)
                .field("Repo", modifier.repo_id.unwrap_or_else(|| "All repos".to_string()), true)
                .field("Priority", modifier.priority.to_string(), true)
                .field("Type", kind, true)
                .field(
                    "Redirect channel",
                    modifier.redirect_channel.map(|c| format!("<#{}>", c)).unwrap_or_else(|| "Default channel".to_string()),
                    true,
                )
                .field("Events", format!("`{}`", modifier.events.join(", ")), false)
                .field("Created at", format!("<t:{}:R>", modifier.created_at.timestamp()), false),
        );
    }

    ctx.send(cr).await?;

    Ok(())
}

#[poise::command(
    slash_command,
    guild_only,
    guild_cooldown = 60,
    required_permissions = "MANAGE_GUILD"
)]
#[allow(clippy::too_many_arguments)]
pub async fn edit(
    ctx: Context<'_>,
    #[description = "The modifier ID"]
    #[autocomplete = "autocomplete::modifier_id"]
    modifier_id: String,
    #[description = "The events to match against, comma/space seperated"] events: Option<String>,
    #[description = "Blacklist the events"] blacklisted: Option<bool>,
    #[description = "Whitelist the events. Other events will not be allowed"] whitelisted: Option<bool>,
    #[description = "Priority. Use 0 for normal priority"] priority: Option<i32>,
    #[description = "Repository ID. Pass \"none\" to match all repos"]
    #[autocomplete = "autocomplete::repo_id"]
    repo_id: Option<String>,
    #[description = "Redirect channel ID"] redirect_channel: Option<ChannelId>,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild_id = ctx.guild_id().unwrap().to_string();

    let existing = sqlx::query!(
        "SELECT webhook_id FROM event_modifiers WHERE id = $1 AND guild_id = $2",
        modifier_id,
        guild_id
    )
    .fetch_optional(&data.pool)
    .await?;

    let Some(existing) = existing else {
        return Err("That modifier doesn't exist!".into());
    };

    let mut tx = data.pool.begin().await?;

    if let Some(events) = events {
        let events = events
            .replace('`', "")
            .replace(',', " ")
            .replace("  ", " ")
            .to_lowercase()
            .split(' ')
            .map(|s| s.to_string())
            .collect::<Vec<String>>();

        sqlx::query!(
            "UPDATE event_modifiers SET events = $1 WHERE id = $2",
            &events,
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(blacklisted) = blacklisted {
        sqlx::query!(
            "UPDATE event_modifiers SET blacklisted = $1 WHERE id = $2",
            blacklisted,
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(whitelisted) = whitelisted {
        sqlx::query!(
            "UPDATE event_modifiers SET whitelisted = $1 WHERE id = $2",
            whitelisted,
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(priority) = priority {
        sqlx::query!(
            "UPDATE event_modifiers SET priority = $1 WHERE id = $2",
            priority,
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(repo_id) = repo_id {
        let parsed_repo_id = if repo_id.is_empty() || repo_id.eq_ignore_ascii_case("none") {
            None
        } else {
            let repo = sqlx::query!(
                "SELECT COUNT(1) FROM repos WHERE id = $1 AND webhook_id = $2",
                repo_id,
                existing.webhook_id
            )
            .fetch_one(&mut *tx)
            .await?;

            if repo.count.unwrap_or_default() == 0 {
                return Err("That repo doesn't exist on this webhook. Use `/newrepo` to link it first.".into());
            }

            Some(repo_id)
        };

        sqlx::query!(
            "UPDATE event_modifiers SET repo_id = $1 WHERE id = $2",
            parsed_repo_id,
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(redirect_channel) = redirect_channel {
        sqlx::query!(
            "UPDATE event_modifiers SET redirect_channel = $1 WHERE id = $2",
            redirect_channel.to_string(),
            modifier_id
        )
        .execute(&mut *tx)
        .await?;
    }

    sqlx::query!(
        "UPDATE event_modifiers SET last_updated_by = $1 WHERE id = $2",
        ctx.author().id.to_string(),
        modifier_id
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Modifier updated successfully!").await?;

    Ok(())
}
