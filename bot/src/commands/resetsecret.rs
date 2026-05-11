use poise::serenity_prelude::CreateMessage;
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error};

/// Resets a webhook secret. DMs must be open
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn resetsecret(
    ctx: Context<'_>,
    #[description = "The webhook ID"]
    #[autocomplete = "super::autocomplete_webhooks"]
    id: String,
) -> Result<(), Error> {
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        &ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;
    
    if guild.count.unwrap_or_default() == 0 {
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` (or ``git!newhook``) to create one".into());
    }

    let webhook = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
        &id,
        &ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    if webhook.count.unwrap_or_default() == 0 {
        return Err("That webhook doesn't exist! Use ``/newhook`` (or ``git!newhook``) to create one".into());
    }

    let webh_secret = Alphanumeric.sample_string(&mut rand::thread_rng(), 256);

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
        &id,
        &ctx.guild_id().unwrap().to_string()
    )
    .execute(&data.pool)
    .await?;

    dm.id.send_message(
        &ctx,
        CreateMessage::new()
        .content(
            format!(
                "Your new webhook secret is `{webh_secret}`. 

Update this webhooks information in your GitHub/GitLab settings now. Your webhook will not accept messages unless you do so!

**Delete this message after you're done!**",
                webh_secret=webh_secret
            )    
        )
    ).await?;

    ctx.say("Webhook secret updated! Check your DMs for the webhook information.").await?;
    
    Ok(())
}
