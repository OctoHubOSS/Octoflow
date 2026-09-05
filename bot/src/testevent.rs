//  Copyright (C) 2026 NodeByte LTD

use poise::serenity_prelude::{CreateEmbed, CreateEmbedAuthor, CreateEmbedFooter};
use poise::CreateReply;
use serde::Deserialize;

use crate::{autocomplete, config, Context, Error};

#[derive(Deserialize, Default)]
struct SimulateResponse {
    #[serde(default)]
    acl_fail: String,
    #[serde(default)]
    channels: Vec<String>,
    #[serde(default)]
    embeds: Vec<SimEmbed>,
}

#[derive(Deserialize, Default)]
struct SimEmbed {
    #[serde(default)]
    title: String,
    #[serde(default)]
    description: String,
    #[serde(default)]
    url: String,
    #[serde(default)]
    color: i64,
    #[serde(default)]
    author: Option<SimAuthor>,
    #[serde(default)]
    footer: Option<SimFooter>,
    #[serde(default)]
    fields: Vec<SimField>,
}

#[derive(Deserialize)]
struct SimAuthor {
    #[serde(default)]
    name: String,
    #[serde(default)]
    icon_url: String,
}

#[derive(Deserialize)]
struct SimFooter {
    #[serde(default)]
    text: String,
}

#[derive(Deserialize)]
struct SimField {
    name: String,
    value: String,
    #[serde(default)]
    inline: bool,
}

#[poise::command(slash_command, guild_only, required_permissions = "MANAGE_GUILD")]
pub async fn testevent(
    ctx: Context<'_>,
    #[description = "The webhook ID"]
    #[autocomplete = "autocomplete::webhook_id"]
    webhook_id: String,
    #[description = "The repo ID to simulate for"]
    #[autocomplete = "autocomplete::repo_id"]
    repo_id: String,
    #[description = "Which GitHub event to simulate"]
    #[choices("push", "pull_request", "issues", "issue_comment", "release", "star", "fork", "ping")]
    event: &'static str,
) -> Result<(), Error> {
    ctx.defer_ephemeral().await?;

    let Some(secret) = config::CONFIG.dashboard_internal_secret.clone() else {
        ctx.say("`/testevent` isn't configured on this bot yet (missing `dashboard_internal_secret` in config.yaml).").await?;
        return Ok(());
    };

    let api_url = config::CONFIG.api_url[0].clone();
    let client = reqwest::Client::new();

    let result = client
        .post(format!("{}/api/dashboard/webhooks/{}/simulate", api_url, webhook_id))
        .header("X-Internal-Secret", secret)
        .json(&serde_json::json!({
            "guild_id": ctx.guild_id().unwrap().to_string(),
            "repo_id": repo_id,
            "event": event,
        }))
        .send()
        .await?;

    if !result.status().is_success() {
        let status = result.status();
        let body = result.text().await.unwrap_or_default();
        ctx.send(
            CreateReply::default()
                .ephemeral(true)
                .content(format!("Simulation failed ({}): {}", status, body)),
        )
        .await?;
        return Ok(());
    }

    let sim: SimulateResponse = result.json().await?;

    if !sim.acl_fail.is_empty() {
        ctx.send(CreateReply::default().ephemeral(true).content(format!(
            "This event would be **blocked** before it reached a channel: `{}`",
            sim.acl_fail
        )))
        .await?;
        return Ok(());
    }

    if sim.embeds.is_empty() {
        ctx.send(
            CreateReply::default()
                .ephemeral(true)
                .content("No embed was produced - nothing to preview."),
        )
        .await?;
        return Ok(());
    }

    let channels_text = if sim.channels.is_empty() {
        "No channel is configured for this repo.".to_string()
    } else {
        sim.channels.iter().map(|c| format!("<#{}>", c)).collect::<Vec<_>>().join(", ")
    };

    let mut reply = CreateReply::default().ephemeral(true).content(format!(
        "**Preview only - nothing was actually sent.**\nWould send to: {}",
        channels_text
    ));

    for e in sim.embeds {
        let mut embed = CreateEmbed::new().title(e.title).description(e.description);

        if e.color != 0 {
            embed = embed.color(e.color as u32);
        }
        if !e.url.is_empty() {
            embed = embed.url(e.url);
        }
        if let Some(author) = e.author {
            let mut a = CreateEmbedAuthor::new(author.name);
            if !author.icon_url.is_empty() {
                a = a.icon_url(author.icon_url);
            }
            embed = embed.author(a);
        }
        if let Some(footer) = e.footer {
            if !footer.text.is_empty() {
                embed = embed.footer(CreateEmbedFooter::new(footer.text));
            }
        }
        for f in e.fields {
            embed = embed.field(f.name, f.value, f.inline);
        }

        reply = reply.embed(embed);
    }

    ctx.send(reply).await?;

    Ok(())
}
