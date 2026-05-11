use crate::{Context, Error};

/// Deletes a repository
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn delrepo(
    ctx: Context<'_>,
    #[description = "The repo ID"]
    #[autocomplete = "super::autocomplete_repos"]
    id: String,
) -> Result<(), Error> { 
    let data = ctx.data();

    sqlx::query!(
        "DELETE FROM repos WHERE id = $1 AND guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).execute(&data.pool).await?;

    ctx.say("Repo deleted!").await?;
    Ok(())
}
