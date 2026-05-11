use poise::serenity_prelude::ChannelId;
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error};

/// Creates a new repository for a webhook
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newrepo(
    ctx: Context<'_>,
    #[description = "The webhook ID to use"]
    #[autocomplete = "super::autocomplete_webhooks"]
    webhook_id: String,
    #[description = "The repo owner or organization"] owner: String,
    #[description = "The repo name"] name: String,
    #[description = "The channel to send to"] channel: ChannelId,
) -> Result<(), Error> { 
    let data = ctx.data();

    // Check if the guild exists on our DB
    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        &ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;
    
    if guild.count.unwrap_or_default() == 0 {
        // If it doesn't, return a error
        return Err("You don't have any webhooks in this guild! Use ``/newhook`` (or ``git!newhook``) to create one".into());
    }

    // Check webhook count
    let webhook_count = sqlx::query!(
        "SELECT COUNT(1) FROM webhooks WHERE guild_id = $1",
        &ctx.guild_id().unwrap().to_string()
    )
    .fetch_one(&data.pool)
    .await?;

    let count = webhook_count.count.unwrap_or_default();

    if count == 0 {
        Err("You don't have any webhooks in this guild! Use ``/newhook`` (or ``git!newhook``) to create one".into())
    } else {
        // Check if the webhook exists
        let webhook = sqlx::query!(
            "SELECT COUNT(1) FROM webhooks WHERE id = $1 AND guild_id = $2",
            &webhook_id,
            &ctx.guild_id().unwrap().to_string()
        )
        .fetch_one(&data.pool)
        .await?;

        if webhook.count.unwrap_or_default() == 0 {
            return Err("That webhook doesn't exist! Use ``/newhook`` (or ``git!newhook``) to create one".into());
        }

        let repo_name = (owner+"/"+&name).to_lowercase();

        // Get provider to validate repo
        let provider_query = sqlx::query!(
            "SELECT provider FROM webhooks WHERE id = $1 AND guild_id = $2",
            &webhook_id,
            &ctx.guild_id().unwrap().to_string()
        )
        .fetch_one(&data.pool)
        .await?;

        let provider = provider_query.provider.unwrap_or_else(|| "github".to_string());

        // Validate repository exists
        let client = reqwest::Client::new();
        let exists = if provider == "gitlab" {
            // GitLab API check
            let url = format!("https://gitlab.com/api/v4/projects/{}", urlencoding::encode(&repo_name));
            let res = client.get(&url).send().await;
            
            if let Ok(response) = res {
                response.status().is_success()
            } else {
                false
            }
        } else {
            // GitHub API check
            let url = format!("https://api.github.com/repos/{}", repo_name);
            let res = client.get(&url)
                .header("User-Agent", "OctoFlow-Discord-Bot")
                .send().await;
                
            if let Ok(response) = res {
                response.status().is_success()
            } else {
                false
            }
        };

        if !exists {
            return Err("That repository could not be found! Make sure it exists and is public (or use your custom GitLab URL if self-hosted, though validation only works for public repos on github.com/gitlab.com currently).".into());
        }

        // Check if the repo exists
        let repo = sqlx::query!(
            "SELECT COUNT(1) FROM repos WHERE lower(repo_name) = $1 AND webhook_id = $2",
            &repo_name,
            &webhook_id
        )
        .fetch_one(&data.pool)
        .await?;

        if repo.count.unwrap_or_default() == 0 {
            // If it doesn't, create it
            let id = Alphanumeric.sample_string(&mut rand::thread_rng(), 32);

            sqlx::query!(
                "INSERT INTO repos (id, webhook_id, repo_name, channel_id, guild_id, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
                id,
                &webhook_id,
                &repo_name,
                channel.to_string(),
                &ctx.guild_id().unwrap().to_string(),
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
            Err("That repo already exists! Use ``/delrepo`` (or ``git!delrepo``) to delete it".into())
        }
    }
}
