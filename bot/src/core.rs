use log::error;
use poise::{serenity_prelude::{CreateMessage, ChannelId, CreateEmbed}, CreateReply};
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error, config};

/// Lsts all webhooks in a guild with their respective repos and channel IDs
#[poise::command(slash_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn list(
    ctx: Context<'_>,
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
        // If it doesn't, return an error
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;

        ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook``").await?;
    } else {
        // Get all webhooks
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
                        "No repos yet. Use ``/newrepo`` to add one".to_string()
                    } else {
                        repos
                            .iter()
                            .map(|r| format!("``{}`` {} -> <#{}>", r.id, r.repo_name, r.channel_id))
                            .collect::<Vec<String>>()
                            .join("\n")
                    };

                    cr = cr.embed(
                        CreateEmbed::new()
                        .title(format!("Webhook \"{}\"", webhook.comment))
                        .field("Webhook ID", webhook_id.clone(), false)
                        .field("Hook URL (visit for hook info, add to Github to recieve events)", api_url.clone()+"/kittycat?id="+&webhook_id, false)
			.field("Marked as Broken", format!("{}", webhook.broken), false)
                        .field("Created at", webhook.created_at.to_string(), false)
                        .field("Repos", repos_field, false)
                    );
                };

                ctx.send(cr).await?;
            },
            Err(e) => {
                error!("Error fetching webhooks: {:?}", e);
                ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook``").await?;
            }
        }
    }

    Ok(())
}

/// Creates a new webhook in a guild for sending Github notifications
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newhook(
    ctx: Context<'_>,
    #[description = "The comment for the webhook"] comment: String,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
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
        // If it doesn't, create it
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;
    }

    // Check webhook count
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook_count.count.unwrap_or_default() >= 5 {
        ctx.say("You can't have more than 5 webhooks per guild").await?;
        return Ok(());
    }

    // Create the webhook
    let id = Alphanumeric.sample_string(&mut rand::thread_rng(), 32);

    let webh_secret = Alphanumeric.sample_string(&mut rand::thread_rng(), 256);

    // Create a new dm channel with the user if not slash command
    let dm_channel = ctx.author().create_dm_channel(ctx.http()).await;

    let dm = match dm_channel {
        Ok(dm) => dm,
        Err(_) => {
            ctx.say("I couldn't create a DM channel with you, please enable DMs from server members").await?;
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

    dm.id.send_message(
        &ctx,
        CreateMessage::new()
        .content(
            format!(
                "
Next, add the following webhook to your Github repositories (or organizations): `{api_url}/kittycat?id={id}`

Set the `Secret` field to `{webh_secret}` and ensure that Content Type is set to `application/json`. 

When creating repositories, use `{id}` as the ID.

**Backup domains (replace {api_url} with these if Octoflow fails):** {api_domains}
            
**Note that the above URL and secret is unique and should not be shared with others**

**Delete this message after you're done!**
                ",
                api_url=config::CONFIG.api_url[0],
                api_domains=config::CONFIG.api_url[1..].join(", "),
                id=id,
                webh_secret=webh_secret
            )    
        )
    ).await?;

    ctx.say("Webhook created! Check your DMs for the webhook information.").await?;
    
    Ok(())
}

/// Edits a webhook
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn edithook(
    ctx: Context<'_>,
    #[description = "The webhook ID"] id: String,
    #[description = "The comment for the webhook"] comment: Option<String>,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
    #[description = "The new secret for the webhook"] webhook_secret: Option<String>,
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
        // If it doesn't, create it
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;
    }

    // Check webhook for existence
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1 AND id = $2",
        ctx.guild_id().unwrap().to_string(),
        id
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook_count.count.unwrap_or_default() == 0 {
        ctx.say("This webhook does not exist!").await?;
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

    // Update last_updated_at and last_updated_by regardless
    sqlx::query!(
        "UPDATE webhooks SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3",
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Webhook updated successfully!").await?;
    
    Ok(())
}

/// Creates a new repository for a webhook
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newrepo(
    ctx: Context<'_>,
    #[description = "The webhook ID to use"] webhook_id: String,
    #[description = "The repo owner or organization"] owner: String,
    #[description = "The repo name"] name: String,
    #[description = "The channel to send to"] channel: ChannelId,
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

    // Check webhook count
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    let count = webhook_count.count.unwrap_or_default();

    if count == 0 {
        Err("You don't have any webhooks in this guild! Use ``/newhook`` to create one".into())
    } else {
        // Check if the webhook exists
        let webhook = sqlx::query!(
            "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
            webhook_id,
            ctx.guild_id().unwrap().to_string()
        )
        .fetch_one(&data.pool)
        .await?;

        if webhook.count.unwrap_or_default() == 0 {
            return Err("That webhook doesn't exist! Use ``/newhook`` to create one".into());
        }

        let repo_name = (owner+"/"+&name).to_lowercase();

        // Check if the repo exists
        let repo = sqlx::query!(
            "SELECT COUNT(1) FROM repos WHERE lower(repo_name) = $1 AND webhook_id = $2",
            &repo_name,
            webhook_id
        )
        .fetch_one(&data.pool)
        .await?;

        if repo.count.unwrap_or_default() == 0 {
            // If it doesn't, create it
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
                format!("Repository created with ID of ``{id}``!", id=id)
            ).await?;

            Ok(())
        } else {
            Err("That repo already exists! Use ``/delrepo`` to delete it".into())
        }
    }
}

/// Deletes a webhook
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn delhook(
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

    sqlx::query!(
        "DELETE FROM webhooks WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    ctx.say("Webhook deleted if it exists!").await?;

    Ok(())
}

/// Deletes a repository
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

    ctx.say("Repo deleted!").await?;

    Ok(())
}

/// Updates the channel for a repository
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn setrepochannel(
    ctx: Context<'_>,
    #[description = "The repo ID"] id: String,
    #[description = "The new channel ID"] channel: ChannelId,
) -> Result<(), Error> { 
    let data = ctx.data();

    // Check if the repo exists
    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist! Use ``/newrepo`` to create one".into());
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

    ctx.say("Channel updated!").await?;

    Ok(())
}

/// Resets a webhook secret. DMs must be open
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn resetsecret(
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
        return Err("That webhook doesn't exist! Use ``/newhook`` to create one".into());
    }

    let webh_secret = Alphanumeric.sample_string(&mut rand::thread_rng(), 256);

    // Try to DM the user
    // Create a new dm channel with the user if not slash command
    let dm_channel = ctx.author().create_dm_channel(ctx.http()).await;

    let dm = match dm_channel {
        Ok(dm) => dm,
        Err(_) => {
            ctx.say("I couldn't create a DM channel with you, please enable DMs from server members").await?;
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
        CreateMessage::new()
        .content(
            format!(
                "Your new webhook secret is `{webh_secret}`. 

Update this webhooks information in GitHub settings now. Your webhook will not accept messages from GitHub unless you do so!

**Delete this message after you're done!**
                ",
                webh_secret=webh_secret
            )    
        )
    ).await?;

    ctx.say("Webhook secret updated! Check your DMs for the webhook information.").await?;

    Ok(())
}

/// Edits a repository's name and/or channel. Only the fields you provide are changed
#[poise::command(slash_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn editrepo(
    ctx: Context<'_>,
    #[description = "The repo ID"] id: String,
    #[description = "The new repo owner/name (e.g. octocat/Hello-World)"] repo_name: Option<String>,
    #[description = "The new channel to send to"] channel: Option<ChannelId>,
) -> Result<(), Error> {
    let data = ctx.data();

    // Check if the repo exists
    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist! Use ``/newrepo`` to create one".into());
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

    // Update last_updated_at and last_updated_by regardless
    sqlx::query!(
        "UPDATE repos SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3",
        ctx.author().id.to_string(),
        id,
        ctx.guild_id().unwrap().to_string()
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Repo updated successfully!").await?;

    Ok(())
}
