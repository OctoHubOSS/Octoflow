use poise::serenity_prelude::ChannelId;
use rand::distributions::{Alphanumeric, DistString};

use crate::{Context, Error};

/// Creates a new repository for a webhook
#[poise::command(slash_command, prefix_command, guild_only, guild_cooldown = 60, required_permissions = "MANAGE_GUILD")]
pub async fn newrepo(
    ctx: Context<'_>,
    #[description = "Webhook ID from `/list`"]
    webhook_id: String,
    #[description = "The repo owner or organization"] owner: String,
    #[description = "The repo name"] name: String,
    #[description = "The channel to send to"] channel: ChannelId,
) -> Result<(), Error> {
    ctx.defer().await?;

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
