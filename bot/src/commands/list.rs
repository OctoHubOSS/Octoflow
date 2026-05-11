use log::error;
use poise::{serenity_prelude::CreateEmbed, CreateReply};

use crate::{Context, Error, config};

/// Lists all webhooks in a guild with their respective repos and channel IDs
#[poise::command(slash_command, prefix_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn list(
    ctx: Context<'_>,
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
        // If it doesn't, create it
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            &ctx.guild_id().unwrap().to_string()
        )
        .execute(&data.pool)
        .await?;

        ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook`` (or ``git!newhook``)").await?;
    } else {
        // Get all webhooks
        let webhooks = sqlx::query!(
            "SELECT id, broken, comment, created_at, COALESCE(provider, 'github') as provider FROM webhooks WHERE guild_id = $1",
            &ctx.guild_id().unwrap().to_string()
        )
        .fetch_all(&data.pool)
        .await;

        match webhooks {
            Ok(webhooks) => {
                if webhooks.is_empty() {
                    ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook`` (or ``git!newhook``)").await?;
                    return Ok(());
                }

                let mut cr = CreateReply::default()
                .content("Here are all the webhooks in this guild:");

                let api_url = config::CONFIG.api_url[0].clone();

                for webhook in webhooks {
                    let webhook_id = webhook.id;
                    let provider = webhook.provider.unwrap_or_else(|| "github".to_string());
                    let provider_label = if provider == "gitlab" { "GitLab" } else { "GitHub" };

                    cr = cr.embed(
                        CreateEmbed::new()
                        .title(format!("Webhook \"{}\"", webhook.comment))
                        .field("Webhook ID", webhook_id.clone(), false)
                        .field("Hook URL", format!("`{}/kittycat?id={}`", api_url, webhook_id), false)
                        .field("Provider", provider_label.to_string(), true)
                        .field("Marked as Broken", format!("{}", webhook.broken), true)
                        .field("Created at", webhook.created_at.to_string(), true)
                    );
                };

                ctx.send(cr).await?;
            },
            Err(e) => {
                error!("Error fetching webhooks: {:?}", e);
                ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook`` (or ``git!newhook``)").await?;
            }
        }
    }

    Ok(())
}
