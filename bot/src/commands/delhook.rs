use crate::{Context, Error};

/// Deletes a webhook
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn delhook(
    ctx: Context<'_>,
    #[description = "The webhook ID"]
    #[autocomplete = "super::autocomplete_webhooks"]
    id: String,
) -> Result<(), Error> { 
    let data = ctx.data();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        &ctx.guild_id().unwrap().to_string()
    ).fetch_one(&data.pool).await?;
    
    if guild.count.unwrap_or_default() == 0 {
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` (or ``git!newhook``) to create one".into());
    }

    sqlx::query!(
        "DELETE FROM webhooks WHERE id = $1 AND guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).execute(&data.pool).await?;

    ctx.say("Webhook deleted if it exists!").await?;
    Ok(())
}
