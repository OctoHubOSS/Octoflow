pub mod list;
pub mod newhook;
pub mod edithook;
pub mod newrepo;
pub mod editrepo;
pub mod delhook;
pub mod delrepo;
pub mod setrepochannel;
pub mod resetsecret;

use crate::{Context, Error};

pub(crate) async fn autocomplete_webhooks<'a>(
    ctx: Context<'a>,
    partial: &'a str,
) -> impl Iterator<Item = String> + 'a {
    let data = ctx.data();
    let guild_id = match ctx.guild_id() {
        Some(id) => id.to_string(),
        None => return vec![].into_iter(),
    };

    struct WebhookChoice {
        id: String,
        comment: String,
    }

    let webhooks = match sqlx::query_as!(
        WebhookChoice,
        "SELECT id, comment FROM webhooks WHERE guild_id = $1",
        guild_id
    )
    .fetch_all(&data.pool)
    .await {
        Ok(v) => v,
        Err(_) => return vec![].into_iter(),
    };

    webhooks
        .into_iter()
        .filter(move |w| {
            w.comment.to_lowercase().contains(&partial.to_lowercase())
                || w.id.to_lowercase().contains(&partial.to_lowercase())
        })
        .map(|w| w.id)
        .collect::<Vec<_>>()
        .into_iter()
}

pub(crate) async fn autocomplete_repos<'a>(
    ctx: Context<'a>,
    partial: &'a str,
) -> impl Iterator<Item = String> + 'a {
    let data = ctx.data();
    let guild_id = match ctx.guild_id() {
        Some(id) => id.to_string(),
        None => return vec![].into_iter(),
    };

    struct RepoChoice {
        id: String,
        repo_name: String,
    }

    let repos = match sqlx::query_as!(
        RepoChoice,
        "SELECT id, repo_name FROM repos WHERE guild_id = $1",
        guild_id
    )
    .fetch_all(&data.pool)
    .await {
        Ok(v) => v,
        Err(_) => return vec![].into_iter(),
    };

    repos
        .into_iter()
        .filter(move |r| {
            r.repo_name.to_lowercase().contains(&partial.to_lowercase())
                || r.id.to_lowercase().contains(&partial.to_lowercase())
        })
        .map(|r| r.id)
        .collect::<Vec<_>>()
        .into_iter()
}
