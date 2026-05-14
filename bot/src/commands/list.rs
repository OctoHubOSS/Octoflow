use log::error;
use poise::serenity_prelude::{Colour, CreateEmbed, CreateEmbedFooter};
use poise::CreateReply;

use crate::{config, Context, Error};

/// Discord allows at most 10 embeds per message.
const MAX_EMBEDS_PER_MESSAGE: usize = 10;

/// Lists all webhooks in a guild with their respective repos and channel IDs
#[poise::command(slash_command, prefix_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn list(ctx: Context<'_>) -> Result<(), Error> {
    ctx.defer().await?;

    let data = ctx.data();
    let guild_id = ctx.guild_id().unwrap().to_string();

    let guild = sqlx::query!(
        "SELECT COUNT(1) FROM guilds WHERE id = $1",
        &guild_id
    )
    .fetch_one(&data.pool)
    .await?;

    if guild.count.unwrap_or_default() == 0 {
        sqlx::query!(
            "INSERT INTO guilds (id) VALUES ($1)",
            &guild_id
        )
        .execute(&data.pool)
        .await?;

        ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook`` (or ``git!newhook``)").await?;
        return Ok(());
    }

    let webhooks = sqlx::query!(
        "SELECT id, broken, comment, created_at, COALESCE(provider, 'github') as provider FROM webhooks WHERE guild_id = $1",
        &guild_id
    )
    .fetch_all(&data.pool)
    .await;

    match webhooks {
        Ok(webhooks) => {
            if webhooks.is_empty() {
                ctx.say("This guild doesn't have any webhooks yet. Get started with ``/newhook`` (or ``git!newhook``)").await?;
                return Ok(());
            }

            let api_url = config::CONFIG.api_url[0].clone();
            let chunks: Vec<_> = webhooks.chunks(MAX_EMBEDS_PER_MESSAGE).collect();
            let total_parts = chunks.len();

            for (i, chunk) in chunks.into_iter().enumerate() {
                let content = if total_parts == 1 {
                    "Here are all the webhooks in this guild:".to_string()
                } else if i == 0 {
                    format!(
                        "Here are all the webhooks in this guild (part 1 of {}):",
                        total_parts
                    )
                } else {
                    format!(
                        "Webhooks continued (part {} of {}):",
                        i + 1,
                        total_parts
                    )
                };

                let mut cr = CreateReply::default().content(content);

                for webhook in chunk {
                    let webhook_id = webhook.id.clone();
                    let provider = webhook.provider.as_deref().unwrap_or("github");
                    let provider_label = if provider == "gitlab" { "GitLab" } else { "GitHub" };
                    let status_note = if webhook.broken {
                        "Marked broken — use `/resetsecret` or recreate the hook if deliveries fail."
                    } else {
                        "Receiving events normally."
                    };
                    let created_ts = webhook.created_at.timestamp();
                    let broken_label = if webhook.broken { "Yes" } else { "No" };

                    cr = cr.embed(
                        CreateEmbed::new()
                            .colour(Colour::from_rgb(88, 101, 242))
                            .title(webhook.comment.clone())
                            .description(format!(
                                "**{}** webhook · {}\nCreated <t:{}:f>",
                                provider_label, status_note, created_ts
                            ))
                            .field("Webhook ID", format!("`{}`", webhook_id), true)
                            .field("Broken?", broken_label, true)
                            .field(
                                "Endpoint",
                                format!("`{}/kittycat?id={}`", api_url, webhook_id),
                                false,
                            )
                            .footer(CreateEmbedFooter::new("OctoFlow · manage webhooks with /newhook and /edithook")),
                    );
                }

                ctx.send(cr).await?;
            }
        }
        Err(e) => {
            error!("Error fetching webhooks: {:?}", e);
            ctx.say("Could not load webhooks from the database. Try again in a moment.")
                .await?;
        }
    }

    Ok(())
}
