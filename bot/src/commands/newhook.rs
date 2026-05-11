use poise::serenity_prelude::CreateMessage;
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error, config};

/// Creates a new webhook in a guild for sending GitHub/GitLab notifications
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newhook(
    ctx: Context<'_>,
    #[description = "The comment for the webhook"] comment: String,
    #[description = "Provider: github or gitlab"] provider: Option<String>,
    #[description = "Is the webhook broken?"] broken: Option<bool>,
) -> Result<(), Error> {
    let data = ctx.data();

    let provider = provider.unwrap_or_else(|| "github".to_string()).to_lowercase();
    
    if provider != "github" && provider != "gitlab" {
        ctx.say("Invalid provider! Use `github` or `gitlab`").await?;
        return Ok(());
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

    // Check webhook count
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        &ctx.guild_id().unwrap().to_string()
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
        "INSERT INTO webhooks (id, guild_id, comment, secret, broken, provider, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
        id,
        &ctx.guild_id().unwrap().to_string(),
        comment,
        webh_secret,
	    broken.unwrap_or(false),
        provider,
        ctx.author().id.to_string(),
        ctx.author().id.to_string(),
    )
    .execute(&data.pool)
    .await?;

    ctx.say("Webhook created! Trying to DM you the credentials...").await?;

    let backup_domains = if config::CONFIG.api_url.len() > 1 {
        format!("\n**Backup domains:** {}", config::CONFIG.api_url[1..].join(", "))
    } else {
        String::new()
    };

    let dm_content = if provider == "gitlab" {
        format!(
            "\
**GitLab Webhook Setup** 🦊

1. Go to your GitLab project → Settings → Webhooks
2. Set the **URL** to: `{api_url}/kittycat?id={id}`
3. Set the **Secret token** to: `{webh_secret}`
4. Select the events you want to receive
5. Click **Add webhook**

When creating repositories with the bot, use `{id}` as the webhook ID.
{backup_domains}
⚠️ **The above URL and secret is unique — do not share it with others**
🗑️ **Delete this message after you're done!**",
            api_url=config::CONFIG.api_url[0],
            backup_domains=backup_domains,
            id=id,
            webh_secret=webh_secret
        )
    } else {
        format!(
            "\
**GitHub Webhook Setup** 🐙

1. Go to your repo/org → Settings → Webhooks → Add webhook
2. Set the **Payload URL** to: `{api_url}/kittycat?id={id}`
3. Set the **Content type** to `application/json`
4. Set the **Secret** to: `{webh_secret}`
5. Select the events you want to receive
6. Click **Add webhook**

When creating repositories with the bot, use `{id}` as the webhook ID.
{backup_domains}
⚠️ **The above URL and secret is unique — do not share it with others**
🗑️ **Delete this message after you're done!**",
            api_url=config::CONFIG.api_url[0],
            backup_domains=backup_domains,
            id=id,
            webh_secret=webh_secret
        )
    };

    dm.id.send_message(
        &ctx,
        CreateMessage::new()
        .content(dm_content)
    ).await?;

    ctx.say("Webhook created! Check your DMs for the webhook information.").await?;
    
    Ok(())
}
