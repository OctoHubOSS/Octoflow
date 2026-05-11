use crate::{Context, Error};

/// Edits a repository's name
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn editrepo(
    ctx: Context<'_>,
    #[description = "The repo ID"]
    #[autocomplete = "super::autocomplete_repos"]
    id: String,
    #[description = "The new repo owner or organization"] owner: String,
    #[description = "The new repo name"] name: String,
) -> Result<(), Error> { 
    let data = ctx.data();
    let repo_name = (owner+"/"+&name).to_lowercase();

    let repo = sqlx::query!(
        "SELECT COUNT(1) FROM repos WHERE id = $1 AND guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).fetch_one(&data.pool).await?;

    if repo.count.unwrap_or_default() == 0 {
        return Err("That repo doesn't exist!".into());
    }

    let provider_query = sqlx::query!(
        "SELECT webhooks.provider FROM repos JOIN webhooks ON repos.webhook_id = webhooks.id WHERE repos.id = $1 AND repos.guild_id = $2",
        &id, &ctx.guild_id().unwrap().to_string()
    ).fetch_one(&data.pool).await?;

    let provider = provider_query.provider.unwrap_or_else(|| "github".to_string());
    let client = reqwest::Client::new();
    let exists = if provider == "gitlab" {
        let url = format!("https://gitlab.com/api/v4/projects/{}", urlencoding::encode(&repo_name));
        client.get(&url).send().await.map(|r| r.status().is_success()).unwrap_or(false)
    } else {
        let url = format!("https://api.github.com/repos/{}", repo_name);
        client.get(&url).header("User-Agent", "OctoFlow-Discord-Bot").send().await.map(|r| r.status().is_success()).unwrap_or(false)
    };

    if !exists {
        return Err("That repository could not be found! Make sure it exists and is public.".into());
    }

    sqlx::query!(
        "UPDATE repos SET repo_name = $1, last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
        &repo_name, ctx.author().id.to_string(), &id, &ctx.guild_id().unwrap().to_string()
    ).execute(&data.pool).await?;

    ctx.say("Repo name updated successfully!").await?;
    Ok(())
}
