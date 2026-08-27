//  Copyright (C) 2026 NodeByte LTD

use log::error;
use poise::{serenity_prelude::{CreateMessage, ChannelId, CreateEmbedFooter}, CreateReply};
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error, config, embeds};

const NO_WEBHOOKS_MSG: &str = "You don't have any webhooks yet. Run `/newhook` to create one.";

#[poise::command(slash_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn list(
    ctx: Context<'_>,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;

        ctx.say(format!("{} New here? See the [Getting Started guide]({}/getting-started).", NO_WEBHOOKS_MSG, embeds::DOCS_URL)).await?;
    } else {
        let webhooks = sqlx::query!(
            "SELECT id, broken, comment, created_at FROM webhooks WHERE guild_id = $1",
            ctx.guild_id().unwrap().to_string()
        )
        .fetch_all(&data.pool)
        .await;

        match webhooks {
            Ok(webhooks) => {
                let mut cr = CreateReply::default()
                .content("Here are all the webhooks in this guild:");

                let api_url = config::CONFIG.api_url[0].clone();

                for webhook in webhooks {
                    let webhook_id = webhook.id;

                    let repos = sqlx::query!(
                        "SELECT id, repo_name, channel_id FROM repos WHERE webhook_id = $1",
                        webhook_id
                    )
                    .fetch_all(&data.pool)
                    .await?;

                    let repos_field = if repos.is_empty() {
                        "No repos yet. Use `/newrepo` to add one.".to_string()
                    } else {
                        repos
                            .iter()
                            .map(|r| format!("`{}` - {} -> <#{}>", r.id, r.repo_name, r.channel_id))
                            .collect::<Vec<String>>()
                            .join("\n")
                    };

                    let payload_url = format!("{}/kittycat?id={}", api_url, webhook_id);

                    cr = cr.embed(
                        embeds::base()
                        .title("Webhook")
                        .field("Comment", webhook.comment.clone(), false)
                        .field("Webhook ID", format!("`{}`", webhook_id), true)
                        .field("Marked as broken", if webhook.broken { "Yes (won't process events)" } else { "No" }.to_string(), true)
                        .field("Payload URL (paste into GitHub's webhook settings)", format!("```{}```", payload_url), false)
                        .field("Created at", format!("<t:{}:R>", webhook.created_at.and_utc().timestamp()), false)
                        .field("Repos", repos_field, false)
                    );
                };

                ctx.send(cr).await?;
            },
            Err(e) => {
                error!("Error fetching webhooks: {:?}", e);
                ctx.say(format!("{} New here? See the [Getting Started guide]({}/getting-started).", NO_WEBHOOKS_MSG, embeds::DOCS_URL)).await?;
            }
        }
    }

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newhook(
    ctx: Context<'_>,
    #[description = "The comment for the webhook"]
    #[max_length = 200]
    comment: String,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;
    }

    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook_count.count.unwrap_or_default() >= 5 {
        ctx.say("You can't have more than 5 webhooks per server. Delete one with `/delhook` first.").await?;
        return Ok(());
    }

    let id = Alphanumeric.sample_string(&mut rand::thread_rng(), 32);

    let webh_secret = Alphanumeric.sample_string(&mut rand::thread_rng(), 256);

    let dm_channel = ctx.author().create_dm_channel(ctx.http()).await;

    let dm = match dm_channel {
        Ok(dm) => dm,
        Err(_) => {
            ctx.say("I couldn't DM you. Please enable \"Allow direct messages from server members\" in your Privacy Settings and try again. Nothing was created.").await?;
            return Ok(());
        }
    };

    sqlx::query!(
        "INSERT INTO webhooks (id, guild_id, comment, secret, broken, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
        id,
        ctx.guild_id().unwrap().to_string(),
        comment,
        webh_secret,
	broken.unwrap_or(false),
        ctx.author().id.to_string(),
        ctx.author().id.to_string(),
    )
    .execute(&data.pool)
    .await?;

    ctx.say("Webhook created! Trying to DM you the credentials...").await?;

    let api_url = config::CONFIG.api_url[0].clone();
    let payload_url = format!("{}/kittycat?id={}", api_url, id);
    let backup_domains = config::CONFIG.api_url[1..].join(", ");

    let mut embed = embeds::base()
        .footer(CreateEmbedFooter::new("Delete this message once you've copied everything below!"))
        .title("Webhook created - one step left")
        .url(format!("{}/getting-started/github-setup", embeds::DOCS_URL))
        .description(
            "Add this to a GitHub repository or organization: Settings -> Webhooks -> Add webhook. \
            Full walkthrough in the linked guide above if you want screenshots.\n\n\
            The secret below is unique to this webhook, don't share it. Anyone who has it \
            could send fake events to your channels.",
        )
        .field("1. Payload URL", format!("```{}```", payload_url), false)
        .field("2. Content type", "`application/json`", true)
        .field("3. Secret", format!("```{}```", webh_secret), false)
        .field(
            "4. Then in Discord",
            format!("`/newrepo webhook_id:{} owner:... name:... channel:...`", id),
            false,
        )
        .field("Webhook ID", format!("`{}`", id), true);

    if !backup_domains.is_empty() {
        embed = embed.field(
            "Backup Payload URLs (use if the main one is down)",
            backup_domains
                .split(", ")
                .map(|domain| format!("```{}/kittycat?id={}```", domain, id))
                .collect::<Vec<_>>()
                .join(""),
            false,
        );
    }

    dm.id.send_message(&ctx, CreateMessage::new().embed(embed)).await?;

    ctx.say("Webhook created! Check your DMs for the webhook information.").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn edithook(
    ctx: Context<'_>,
    #[description = "The webhook ID"] id: String,
    #[description = "The comment for the webhook"]
    #[max_length = 200]
    comment: Option<String>,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
    #[description = "The new secret for the webhook"] webhook_secret: Option<String>,
    #[description = "Collapse rapid consecutive push events into one summary embed"] batch_events: Option<bool>,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;
    }

    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1 AND id = $2",
        ctx.guild_id().unwrap().to_string(),
        id
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook_count.count.unwrap_or_default() == 0 {
        ctx.say("That webhook doesn't exist. Use `/list` to see your existing webhooks.").await?;
        return Ok(());
    }

    let mut tx = data.pool.begin().await?;

    if let Some(comment) = comment {
        sqlx::query!(
            "UPDATE webhooks SET comment = $1 WHERE id = $2 AND guild_id = $3",
            comment,
            id,
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(broken) = broken {
        sqlx::query!(
            "UPDATE webhooks SET broken = $1 WHERE id = $2 AND guild_id = $3",
            broken,
            id,
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(webhook_secret) = webhook_secret {
        sqlx::query!(
            "UPDATE webhooks SET secret = $1 WHERE id = $2 AND guild_id = $3",
            webhook_secret,
            id,
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(batch_events) = batch_events {
        // Plain (not compile-time-checked) query: batch_events is a brand-new
        // column not yet in the .sqlx offline cache, and there's no live DB
        // handy to regenerate it against. Same runtime behavior either way.
        sqlx::query("UPDATE webhooks SET batch_events = $1 WHERE id = $2 AND guild_id = $3")
            .bind(batch_events)
            .bind(&id)
            .bind(ctx.guild_id().unwrap().to_string())
            .execute(&mut *tx)
            .await?;
    }

    sqlx::query!(
        "UPDATE webhooks SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3",
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Webhook updated.").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newrepo(
    ctx: Context<'_>,
    #[description = "The webhook ID to use"] webhook_id: String,
    #[description = "The repo owner or organization"] owner: String,
    #[description = "The repo name"] name: String,
    #[description = "The channel to send to"] channel: ChannelId,
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

        let repo_name = (owner+"/"+&name).to_lowercase();

        let repo = sqlx::query!(
            "SELECT COUNT(1) FROM repos WHERE lower(repo_name) = $1 AND webhook_id = $2",
            &repo_name,
            webhook_id
        )
        .fetch_one(&data.pool)
        .await?;

        if repo.count.unwrap_or_default() == 0 {

            let id = Alphanumeric.sample_string(&mut rand::thread_rng(), 32);

            sqlx::query!(
                "INSERT INTO repos (id, webhook_id, repo_name, channel_id, guild_id, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
                id,
                webhook_id,
                &repo_name,
                channel.to_string(),
                ctx.guild_id().unwrap().to_string(),
                ctx.author().id.to_string(),
                ctx.author().id.to_string(),
            )
            .execute(&data.pool)
            .await?;

            ctx.say(
                format!("Repository linked! ID: `{id}`. Events for `{repo_name}` will be sent to <#{channel}>.", id=id, repo_name=repo_name, channel=channel)
            ).await?;

            Ok(())
        } else {
            Err("That repo is already linked to this webhook. Use `/editrepo` to change it, or `/delrepo` to remove it first.".into())
        }
    }
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn delhook(
    ctx: Context<'_>,
    #[description = "The webhook ID"] id: String,
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

    sqlx::query!(
        "DELETE FROM webhooks WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    ctx.say("Webhook deleted (if it existed). Its repos and event modifiers were removed too.").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn delrepo(
    ctx: Context<'_>,
    #[description = "The repo ID"] id: String,
) -> Result<(), Error> {
    let data = ctx.data();

    sqlx::query!(
        "DELETE FROM repos WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    ctx.say("Repository deleted (if it existed).").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn setrepochannel(
    ctx: Context<'_>,
    #[description = "The repo ID"] id: String,
    #[description = "The new channel ID"] channel: ChannelId,
) -> Result<(), Error> {
    let data = ctx.data();

    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist. Use `/newrepo` to create one, or `/list` to see your existing repos.".into());
    }

    sqlx::query!(
        "UPDATE repos SET channel_id = $1, last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
        channel.to_string(),
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    ctx.say(format!("Channel updated. Events for this repo now go to <#{}>.", channel)).await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn resetsecret(
    ctx: Context<'_>,
    #[description = "The webhook ID"] id: String,
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

    let webhook = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook.count.unwrap_or_default() == 0 {
        return Err("That webhook doesn't exist. Use `/newhook` to create one, or `/list` to see your existing webhooks.".into());
    }

    let webh_secret = Alphanumeric.sample_string(&mut rand::thread_rng(), 256);

    let dm_channel = ctx.author().create_dm_channel(ctx.http()).await;

    let dm = match dm_channel {
        Ok(dm) => dm,
        Err(_) => {
            ctx.say("I couldn't DM you. Please enable \"Allow direct messages from server members\" in your Privacy Settings and try again. The secret was not rotated.").await?;
            return Ok(());
        }
    };

    sqlx::query!(
        "UPDATE webhooks SET secret = $1, last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
        webh_secret,
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    dm.id.send_message(
        &ctx,
        CreateMessage::new().embed(
            embeds::base()
                .footer(CreateEmbedFooter::new("Delete this message once you've updated GitHub!"))
                .title("Webhook secret rotated")
                .url(format!("{}/getting-started/github-setup", embeds::DOCS_URL))
                .description(
                    "Update the Secret field for this webhook now: GitHub repo/org -> \
                    Settings -> Webhooks -> select this webhook -> Secret. Events will fail \
                    signature verification and be rejected until you do.",
                )
                .field("New secret", format!("```{}```", webh_secret), false)
                .field("Webhook ID", format!("`{}`", id), true),
        ),
    ).await?;

    ctx.say("Webhook secret updated! Check your DMs for the webhook information.").await?;

    Ok(())
}

#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn editrepo(
    ctx: Context<'_>,
    #[description = "The repo ID"] id: String,
    #[description = "The new repo owner/name (e.g. octocat/Hello-World)"] repo_name: Option<String>,
    #[description = "The new channel to send to"] channel: Option<ChannelId>,
    #[description = "Post PR/issue activity into a thread per number instead of the flat channel"] use_threads: Option<bool>,
) -> Result<(), Error> {
    let data = ctx.data();

    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist. Use `/newrepo` to create one, or `/list` to see your existing repos.".into());
    }

    let mut tx = data.pool.begin().await?;

    if let Some(repo_name) = repo_name {
        let repo_name = repo_name.to_lowercase();

        sqlx::query!(
            "UPDATE repos SET repo_name = $1 WHERE id = $2 AND guild_id = $3",
            repo_name,
            id,
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(channel) = channel {
        sqlx::query!(
            "UPDATE repos SET channel_id = $1 WHERE id = $2 AND guild_id = $3",
            channel.to_string(),
            id,
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(use_threads) = use_threads {
        // Same as batch_events above: use_threads is new, not in the .sqlx
        // offline cache yet, no live DB to regenerate it against right now.
        sqlx::query("UPDATE repos SET use_threads = $1 WHERE id = $2 AND guild_id = $3")
            .bind(use_threads)
            .bind(&id)
            .bind(ctx.guild_id().unwrap().to_string())
            .execute(&mut *tx)
            .await?;
    }

    sqlx::query!(
        "UPDATE repos SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3",
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Repository updated.").await?;

    Ok(())
}
