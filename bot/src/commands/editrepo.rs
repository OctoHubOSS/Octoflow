use crate::{Context, Error};

/// Edits a repository's name
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn editrepo(
    ctx: Context<'_>,
    #[description = "Repo ID from `/list`"]
    id: String,
    #[description = "The new repo owner or organization"] owner: String,
    #[description = "The new repo name"] name: String,
) -> Result<(), Error> {
    ctx.defer().await?;

    let data = ctx.data();
    let repo_name = (owner+"/"+&name).to_lowercase();

    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).fetch_one(&data.pool).await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist!".into());
    }

    sqlx::query!(
        "UPDATE repos SET repo_name = $1, last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
        &repo_name, ctx.author().id.to_string(), &id, &ctx.guild_id().unwrap().to_string()
    ).execute(&data.pool).await?;

    ctx.say("Repo name updated successfully!").await?;
    Ok(())
}
