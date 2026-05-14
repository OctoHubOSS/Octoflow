use poise::serenity_prelude::ChannelId;

use crate::{Context, Error};

/// Updates the channel for a repository
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn setrepochannel(
    ctx: Context<'_>,
    #[description = "Repo ID from `/list`"]
    id: String,
    #[description = "The new channel ID"] channel: ChannelId,
) -> Result<(), Error> { 
    let data = ctx.data();

    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).fetch_one(&data.pool).await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist! Use ``/newrepo`` (or ``git!newrepo``) to create one".into());
    }

    sqlx::query!(
        "UPDATE repos SET channel_id = $1, last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
        channel.to_string(), ctx.author().id.to_string(), &id, &ctx.guild_id().unwrap().to_string()
    ).execute(&data.pool).await?;

    ctx.say("Channel updated!").await?;
    Ok(())
}
