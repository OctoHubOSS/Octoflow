use poise::{
    serenity_prelude::{Attachment, CreateAttachment},
    CreateReply,
};
use serde::{Deserialize, Serialize};

use crate::{Context, Error};

const PROTOCOL: u8 = 2;

#[derive(Serialize, Deserialize)]
struct Repo {
    repo_id: String,
    repo_name: String,
    channel_id: String,
}

#[derive(Serialize, Deserialize)]
struct EventModifier {
    event_modifier_id: String,
    repo_id: Option<String>,
    events: Vec<String>,
    blacklisted: bool,
    whitelisted: bool,
    redirect_channel: Option<String>,
    priority: i32,
}

#[derive(Serialize, Deserialize)]
struct Backup {
    protocol: u8,
    event_modifiers: Vec<EventModifier>,
    repos: Vec<Repo>,
}

#[derive(Serialize, Deserialize)]
struct ProtocolCheck {
    protocol: Option<u8>,
}

/// Backups the repositories of a webhook to a JSON file
#[poise::command(
    slash_command,
    guild_only,
    required_permissions = "MANAGE_GUILD"
)]
pub async fn backup(
    ctx: Context<'_>,
    #[description = "The webhook ID"] id: String,
) -> Result<(), Error> {
    let data = ctx.data();

    // Check if the guild exists on our DB
    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        // If it doesn't, return a error
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` to create one".into());
    }

    // Check if the webhook exists
    let webhook = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook.count.unwrap_or_default() == 0 {
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` to create one".into());
    }

    let rows = sqlx::query!(
        "SELECT id, repo_name, channel_id FROM repos WHERE webhook_id = $1",
        id
    )
    .fetch_all(&data.pool)
    .await?;

    let mut repos = Vec::new();

    for row in rows {
        repos.push(Repo {
            repo_id: row.id,
            repo_name: row.repo_name,
            channel_id: row.channel_id,
        });
    }

    // Fetch the event modifiers
    let rows = sqlx::query!(
        "SELECT id, repo_id, events, blacklisted, whitelisted, redirect_channel, priority FROM event_modifiers WHERE webhook_id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string(),
    )
    .fetch_all(&data.pool)
    .await?;

    let mut event_modifiers = Vec::new();

    for row in rows {
        event_modifiers.push(EventModifier {
            event_modifier_id: row.id,
            repo_id: row.repo_id,
            events: row.events,
            blacklisted: row.blacklisted,
            whitelisted: row.whitelisted,
            redirect_channel: row.redirect_channel,
            priority: row.priority,
        });
    }

    let json = serde_json::to_string(&Backup {
        protocol: PROTOCOL,
        event_modifiers,
        repos,
    })?;

    let msg = CreateReply::default()
        .content("Here's your backup file!")
        .attachment(CreateAttachment::bytes(json.into_bytes(), id + ".glb"));

    ctx.send(msg).await?;

    Ok(())
}

/// Restore a created backup to a webhook
#[poise::command(category = "Backups", slash_command, guild_only)]
pub async fn restore(
    ctx: Context<'_>,
    #[description = "The webhook ID to restore the backup to"] id: String,
    #[description = "The backup file"] file: Attachment,
) -> Result<(), Error> {
    let data = ctx.data();

    // Check if the guild exists on our DB
    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        // If it doesn't, return a error
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` to create one".into());
    }

    // Check if the webhook exists
    let webhook = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook.count.unwrap_or_default() == 0 {
        return Err(
            "That webhook doesn't exist! Use ``/newhook`` to create one"
                .into(),
        );
    }

    let backup_bytes = file.download().await?;

    let backup_protocol: ProtocolCheck = serde_json::from_slice(&backup_bytes)?;

    if backup_protocol.protocol.unwrap_or_default() != PROTOCOL {
        return Err(format!(
            "This backup file is not compatible with this version of the bot. 

Protocol version expected: {},
Protocol version found: {}                

Please contact our support team.
            ",
            PROTOCOL,
            backup_protocol.protocol.unwrap_or_default()
        )
        .into());
    }

    let backup: Backup = serde_json::from_slice(&backup_bytes)?;

    // Restore the repositories
    let status = ctx.say("Restoring repositories [1/2]...").await?;

    let mut inserted_repos = 0;
    let mut updated_repos = 0;

    for repo in backup.repos {
        // Check that the repo exists
        let repo_exists = sqlx::query!(
            "SELECT COUNT(1) FROM repos WHERE id = $1 AND webhook_id = $2 AND guild_id = $3",
            repo.repo_id,
            id,
            ctx.guild_id().unwrap().to_string(),
        )
        .fetch_one(&data.pool)
        .await?;

        if repo_exists.count.unwrap_or_default() == 0 {
            // If it doesn't, create it
            sqlx::query!(
                "INSERT INTO repos (id, repo_name, webhook_id, guild_id, channel_id, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
                repo.repo_id,
                repo.repo_name,
                id,
                ctx.guild_id().unwrap().to_string(),
                repo.channel_id,
                ctx.author().id.to_string(),
                ctx.author().id.to_string(),
            )
            .execute(&data.pool)
            .await?;

            inserted_repos += 1;
        } else {
            // If it does, update it
            sqlx::query!(
                "UPDATE repos SET repo_name = $1, channel_id = $2 WHERE id = $3 AND webhook_id = $4 AND guild_id = $5",
                repo.repo_name,
                repo.channel_id,
                repo.repo_id,
                id,
                ctx.guild_id().unwrap().to_string(),
            )
            .execute(&data.pool)
            .await?;

            updated_repos += 1;
        }
    }

    // Restore event modifiers
    status
        .edit(
            ctx,
            CreateReply::default().content("Restoring event modifiers [2/2]..."),
        )
        .await?;

    let mut inserted_modifiers = 0;
    let mut updated_modifiers = 0;

    for event_modifier in backup.event_modifiers {
        // Check that the event modifier exists
        let event_modifier_exists = sqlx::query!(
            "SELECT COUNT(1) FROM event_modifiers WHERE id = $1 AND webhook_id = $2 AND guild_id = $3",
            event_modifier.event_modifier_id,
            id,
            ctx.guild_id().unwrap().to_string(),
        )
        .fetch_one(&data.pool)
        .await?;

        if event_modifier_exists.count.unwrap_or_default() == 0 {
            // If it doesn't, create it
            sqlx::query!(
                "INSERT INTO event_modifiers (id, repo_id, events, blacklisted, whitelisted, redirect_channel, webhook_id, guild_id, priority, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
                event_modifier.event_modifier_id,
                event_modifier.repo_id,
                &event_modifier.events,
                event_modifier.blacklisted,
                event_modifier.whitelisted,
                event_modifier.redirect_channel,
                id,
                ctx.guild_id().unwrap().to_string(),
                event_modifier.priority,
                ctx.author().id.to_string(),
                ctx.author().id.to_string(),
            )
            .execute(&data.pool)
            .await?;

            inserted_modifiers += 1;
        } else {
            // If it does, update it
            sqlx::query!(
                "UPDATE event_modifiers SET repo_id = $1, events = $2, blacklisted = $3, whitelisted = $4, redirect_channel = $5, priority = $6, last_updated_by = $7 WHERE id = $8 AND webhook_id = $9 AND guild_id = $10",
                event_modifier.repo_id,
                &event_modifier.events,
                event_modifier.blacklisted,
                event_modifier.whitelisted,
                event_modifier.redirect_channel,
                event_modifier.priority,
                ctx.author().id.to_string(),
                event_modifier.event_modifier_id,
                id,
                ctx.guild_id().unwrap().to_string(),
            )
            .execute(&data.pool)
            .await?;

            updated_modifiers += 1;
        }
    }

    status
        .edit(
            ctx,
            CreateReply::default().content(format!(
                r#"
**Summary**

- **Inserted repos:** {inserted_repos}
- **Updated repos:** {updated_repos}
- **Inserted event modifiers:** {inserted_modifiers}
- **Updated event modifiers:** {updated_modifiers}
"#,
                inserted_repos = inserted_repos,
                updated_repos = updated_repos,
                inserted_modifiers = inserted_modifiers,
                updated_modifiers = updated_modifiers
            )),
        )
        .await?;

    Ok(())
}
