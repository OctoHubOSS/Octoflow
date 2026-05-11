use crate::{Context, Error};

/// Edits a webhook
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn edithook(
    ctx: Context<'_>,
    #[description = "The webhook ID"]
    #[autocomplete = "super::autocomplete_webhooks"]
    id: String,
    #[description = "The comment for the webhook"] comment: Option<String>,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
    #[description = "The new secret for the webhook"] webhook_secret: Option<String>,
    #[description = "Provider: github or gitlab"] provider: Option<String>,
) -> Result<(), Error> {
    let data = ctx.data();

    // Validate provider if provided
    if let Some(ref p) = provider {
        let p_lower = p.to_lowercase();
        if p_lower != "github" && p_lower != "gitlab" {
            ctx.say("Invalid provider! Use `github` or `gitlab`").await?;
            return Ok(());
        }
    }

    // Validate secret isn't too short if provided
    if let Some(ref s) = webhook_secret {
        if s.len() < 16 {
            ctx.say("Webhook secret must be at least 16 characters long for security!").await?;
            return Ok(());
        }
    }

    // Check if the guild exists on our DB
    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        &ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;
    
    if guild.count.unwrap_or_default() == 0 {
        // If it doesn't, create it
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;
    }

    // Check webhook for existence
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1 AND id = $2",
        &ctx.guild_id().unwrap().to_string(),
        &id
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
            &id,
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(broken) = broken {
        sqlx::query!(
            "UPDATE webhooks SET broken = $1 WHERE id = $2 AND guild_id = $3",
            broken,
            &id,
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(webhook_secret) = webhook_secret {
        sqlx::query!(
            "UPDATE webhooks SET secret = $1 WHERE id = $2 AND guild_id = $3",
            webhook_secret,
            &id,
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    if let Some(provider) = provider {
        sqlx::query!(
            "UPDATE webhooks SET provider = $1 WHERE id = $2 AND guild_id = $3",
            provider.to_lowercase(),
            &id,
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&mut *tx)
        .await?;
    }

    // Update last_updated_at and last_updated_by regardless
    sqlx::query!(
        "UPDATE webhooks SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3",
        ctx.author().id.to_string(),
        &id,
        &ctx.guild_id().unwrap().to_string()
    )
    .execute(&mut *tx)
    .await?;

    tx.commit().await?;

    ctx.say("Webhook updated successfully!").await?;
    
    Ok(())
}
