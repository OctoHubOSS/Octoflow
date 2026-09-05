//  Copyright (C) 2026 NodeByte LTD

use std::time::Duration;

use log::{error, info, warn};
use poise::serenity_prelude::{
    self as prelude, FullEvent,
};
use serde::{Deserialize, Serialize};
use sqlx::postgres::PgPoolOptions;
use serenity::gateway::ActivityData;
use std::sync::Arc;

mod help;
mod core;
mod backups;
mod config;
mod eventmods;
mod embeds;
mod autocomplete;
mod testevent;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");

type Error = Box<dyn std::error::Error + Send + Sync>;
type Context<'a> = poise::Context<'a, Data, Error>;
pub struct Data {
    pub pool: sqlx::PgPool,
}

#[poise::command(slash_command, owners_only, hide_in_help)]
async fn register(ctx: Context<'_>) -> Result<(), Error> {
    poise::builtins::register_application_commands_buttons(ctx).await?;
    Ok(())
}

async fn on_error(error: poise::FrameworkError<'_, Data, Error>) {
    match error {
        poise::FrameworkError::Command { error, ctx, .. } => {
            error!("Error in command `{}`: {:?}", ctx.command().name, error,);
            ctx.say(format!(
                "There was an error running this command: {}",
                error
            ))
            .await
            .unwrap();
        }
        poise::FrameworkError::CommandCheckFailed { error, ctx, .. } => {
            error!(
                "[Possible] error in command `{}`: {:?}",
                ctx.command().name,
                error,
            );
            if let Some(error) = error {
                error!("Error in command `{}`: {:?}", ctx.command().name, error,);
                ctx.say(format!(
                    "Whoa there, do you have permission to do this?: {}",
                    error
                ))
                .await
                .unwrap();
            } else {
                ctx.say("You don't have permission to do this but we couldn't figure out why...")
                    .await
                    .unwrap();
            }
        }
        error => {
            if let Err(e) = poise::builtins::on_error(error).await {
                error!("Error while handling error: {}", e);
            }
        }
    }
}

async fn event_listener<'a>(
    ctx: poise::FrameworkContext<'a, Data, Error>,
    event: &FullEvent,
) -> Result<(), Error> {
    match event {
        FullEvent::InteractionCreate { interaction } => {
            info!("Interaction received: {:?}", interaction.id());
        }
        FullEvent::Ready {
            data_about_bot,
        } => {
            tokio::time::sleep(Duration::from_secs(2)).await;

            info!(
                "{} is ready!",
                data_about_bot.user.name
            );

            if let Err(e) = poise::builtins::register_globally(&ctx.serenity_context.http, &ctx.options().commands).await {
                error!("Failed to register application commands: {:?}", e);
            }

            let bot_id = data_about_bot.user.id;
            let omniplex_commands = flatten_omniplex_commands(&ctx.options().commands);
            tokio::spawn(post_omniplex_commands(bot_id, omniplex_commands));
            tokio::spawn(post_omniplex_changelog(bot_id));

            ctx.serenity_context.set_activity(Some(ActivityData::playing("octoflow.ca")));
        }
        _ => {}
    }

    Ok(())
}

#[tokio::main]
async fn main() {
    const MAX_CONNECTIONS: u32 = 3;

    std::env::set_var("RUST_LOG", "bot=info");

    env_logger::init();

    let mut http =
        prelude::HttpBuilder::new(&config::CONFIG.token);

    if let Some(v) = &config::CONFIG.proxy_url {
        info!("Setting proxy url to {}", v);

        let mut headers = reqwest::header::HeaderMap::new();
        headers.insert(
            "X-Upstream-Authorization",
            reqwest::header::HeaderValue::from_str(&format!("Bot {}", config::CONFIG.token))
                .expect("token should be a valid header value"),
        );

        let reqwest_client = reqwest::Client::builder()
            .default_headers(headers)
            .build()
            .expect("failed to build reqwest client");

        http = http.proxy(v).ratelimiter_disabled(true).client(reqwest_client);
    }

    let http = http.build();

    let client_builder =
        prelude::ClientBuilder::new_with_http(
            Arc::new(http), 
            prelude::GatewayIntents::GUILD_MESSAGES | prelude::GatewayIntents::GUILDS | prelude::GatewayIntents::MESSAGE_CONTENT
        );
    
    let data = Data {
        pool: PgPoolOptions::new()
            .max_connections(MAX_CONNECTIONS)
            .connect(&config::CONFIG.database_url)
            .await
            .expect("Could not initialize connection"),
    };

    let heartbeat_pool = data.pool.clone();

    let framework = poise::Framework::new(
        poise::FrameworkOptions {
            initialize_owners: true,
            event_handler: |ctx, event| Box::pin(event_listener(ctx, event)),
            commands: vec![
                register(),
                help::simplehelp(),
                help::help(),
                core::list(),
                core::newhook(),
                core::edithook(),
                core::newrepo(),
                core::editrepo(),
                core::delhook(),
                core::delrepo(),
                core::setrepochannel(),
                core::resetsecret(),
                backups::backup(),
                backups::restore(),
                eventmods::eventmod(),
                testevent::testevent(),
            ],
            pre_command: |ctx| {
                Box::pin(async move {
                    info!(
                        "Executing command {} for user {} ({})...",
                        ctx.command().qualified_name,
                        ctx.author().name,
                        ctx.author().id
                    );
                })
            },
            post_command: |ctx| {
                Box::pin(async move {
                    info!(
                        "Done executing command {} for user {} ({})...",
                        ctx.command().qualified_name,
                        ctx.author().name,
                        ctx.author().id
                    );
                })
            },
            on_error: |error| Box::pin(on_error(error)),
            ..Default::default()
        },
    );

    let mut client = client_builder
        .framework(framework)
        .data(Arc::new(data))
        .await
        .expect("Error creating client");

    tokio::spawn(heartbeat_task(client.cache.clone(), heartbeat_pool));
    tokio::spawn(omniplex_stats_task(client.cache.clone()));

    if let Err(why) = client.start().await {
        error!("Client error: {:?}", why);
    }
}

async fn heartbeat_task(cache: Arc<prelude::Cache>, pool: sqlx::PgPool) {
    let mut interval = tokio::time::interval(Duration::from_secs(60));

    loop {
        interval.tick().await;

        let guild_ids = cache.guilds();
        let guild_count = guild_ids.len() as i32;
        let shard_count = cache.shard_count().get() as i32;

        let mut member_count: i64 = 0;
        for guild_id in &guild_ids {
            if let Some(guild) = cache.guild(*guild_id) {
                member_count += guild.member_count as i64;
            }
        }

        let result = sqlx::query(
            "INSERT INTO bot_heartbeat (id, guild_count, member_count, shard_count, updated_at) \
             VALUES (1, $1, $2, $3, NOW()) \
             ON CONFLICT (id) DO UPDATE SET guild_count = $1, member_count = $2, shard_count = $3, updated_at = NOW()",
        )
        .bind(guild_count)
        .bind(member_count)
        .bind(shard_count)
        .execute(&pool)
        .await;

        if let Err(e) = result {
            error!("Failed to upsert bot heartbeat: {:?}", e);
        }
    }
}

#[derive(Serialize, Clone)]
struct OmniplexCommandInput {
    name: String,
    description: String,
    usage: String,
    category: String,
}

#[derive(Serialize)]
struct OmniplexUpdateCommands {
    commands: Vec<OmniplexCommandInput>,
}

fn truncate_chars(s: &str, max: usize) -> String {
    s.chars().take(max).collect()
}

fn build_usage(qualified_name: &str, parameters: &[poise::CommandParameter<Data, Error>]) -> String {
    let mut usage = format!("/{}", qualified_name);
    for param in parameters {
        if param.required {
            usage.push_str(&format!(" <{}>", param.name));
        } else {
            usage.push_str(&format!(" [{}]", param.name));
        }
    }
    truncate_chars(&usage, 200)
}

fn flatten_omniplex_commands(commands: &[poise::Command<Data, Error>]) -> Vec<OmniplexCommandInput> {
    let mut out = Vec::new();

    for cmd in commands {
        if cmd.hide_in_help {
            continue;
        }

        if cmd.subcommands.is_empty() {
            out.push(OmniplexCommandInput {
                name: truncate_chars(&format!("/{}", cmd.qualified_name), 100),
                description: truncate_chars(cmd.description.as_deref().unwrap_or(""), 500),
                usage: build_usage(&cmd.qualified_name, &cmd.parameters),
                category: truncate_chars(cmd.category.as_deref().unwrap_or("Uncategorized"), 50),
            });
        } else {
            out.extend(flatten_omniplex_commands(&cmd.subcommands));
        }
    }

    out
}

async fn post_omniplex_commands(bot_id: prelude::UserId, commands: Vec<OmniplexCommandInput>) {
    let Some(token) = config::CONFIG.omniplex_token.clone() else {
        return;
    };

    let mut commands = commands;
    commands.truncate(100);
    let count = commands.len();

    let client = reqwest::Client::new();
    let result = client
        .put(format!("https://api.omniplex.gg/bots/{}/commands", bot_id))
        .header("Authorization", format!("Bot {}", token))
        .json(&OmniplexUpdateCommands { commands })
        .send()
        .await;

    match result {
        Ok(resp) if resp.status().is_success() => {
            info!("Posted {} commands to Omniplex", count);
        }
        Ok(resp) => {
            warn!(
                "Omniplex commands post rejected: {} {}",
                resp.status(),
                resp.text().await.unwrap_or_default()
            );
        }
        Err(e) => warn!("Failed to post Omniplex commands: {:?}", e),
    }
}

const CHANGELOG_MD: &str = include_str!("../../CHANGELOG.md");

struct ChangelogEntry {
    version: String,
    title: String,
    content: String,
}

fn latest_changelog_entry() -> Option<ChangelogEntry> {
    let mut lines = CHANGELOG_MD.lines().peekable();

    while let Some(line) = lines.next() {
        let Some(rest) = line.strip_prefix("## [") else { continue };
        let Some(close) = rest.find(']') else { continue };
        let version = &rest[..close];

        if version == "Unreleased" {
            continue;
        }

        let mut body = String::new();
        while let Some(&next) = lines.peek() {
            if next.starts_with("## [") {
                break;
            }
            body.push_str(next);
            body.push('\n');
            lines.next();
        }

        return Some(ChangelogEntry {
            version: version.to_string(),
            title: format!("v{}", version),
            content: body.trim().to_string(),
        });
    }

    None
}

#[derive(Deserialize)]
struct OmniplexChangelogList {
    changelogs: Vec<OmniplexChangelogEntry>,
}

#[derive(Deserialize)]
struct OmniplexChangelogEntry {
    version: String,
}

#[derive(Serialize)]
struct OmniplexCreateChangelog {
    title: String,
    content: String,
    version: String,
}

async fn post_omniplex_changelog(bot_id: prelude::UserId) {
    let Some(token) = config::CONFIG.omniplex_token.clone() else {
        return;
    };

    let Some(entry) = latest_changelog_entry() else {
        warn!("Could not find a released version entry in CHANGELOG.md");
        return;
    };

    let client = reqwest::Client::new();

    let existing = client
        .get(format!("https://api.omniplex.gg/bots/{}/changelogs", bot_id))
        .send()
        .await;

    let already_posted = match existing {
        Ok(resp) if resp.status().is_success() => match resp.json::<OmniplexChangelogList>().await {
            Ok(list) => list.changelogs.iter().any(|c| c.version == entry.version),
            Err(e) => {
                warn!("Failed to parse existing Omniplex changelogs, skipping post to avoid a duplicate: {:?}", e);
                return;
            }
        },
        Ok(resp) => {
            warn!(
                "Failed to fetch existing Omniplex changelogs ({}), skipping post to avoid a duplicate",
                resp.status()
            );
            return;
        }
        Err(e) => {
            warn!("Failed to fetch existing Omniplex changelogs, skipping post to avoid a duplicate: {:?}", e);
            return;
        }
    };

    if already_posted {
        info!("Omniplex changelog for v{} already posted, skipping", entry.version);
        return;
    }

    let payload = OmniplexCreateChangelog {
        title: truncate_chars(&entry.title, 200),
        content: truncate_chars(&entry.content, 10000),
        version: truncate_chars(&entry.version, 50),
    };

    let result = client
        .post(format!("https://api.omniplex.gg/bots/{}/changelogs", bot_id))
        .header("Authorization", format!("Bot {}", token))
        .json(&payload)
        .send()
        .await;

    match result {
        Ok(resp) if resp.status().is_success() => {
            info!("Posted changelog v{} to Omniplex", entry.version);
        }
        Ok(resp) => {
            warn!(
                "Omniplex changelog post rejected: {} {}",
                resp.status(),
                resp.text().await.unwrap_or_default()
            );
        }
        Err(e) => warn!("Failed to post Omniplex changelog: {:?}", e),
    }
}

#[derive(Serialize)]
struct OmniplexStats {
    servers: u64,
    shards: u64,
    users: u64,
    shard_list: Vec<u64>,
    status: &'static str,
}

async fn omniplex_stats_task(cache: Arc<prelude::Cache>) {
    let Some(token) = config::CONFIG.omniplex_token.clone() else {
        info!("omniplex_token not set, skipping Omniplex stats reporting");
        return;
    };

    let client = reqwest::Client::new();
    let mut interval = tokio::time::interval(Duration::from_secs(5 * 60));

    loop {
        interval.tick().await;

        let guild_ids = cache.guilds();
        let servers = guild_ids.len() as u64;
        let shards: u64 = cache.shard_count().get().into();

        let mut users: u64 = 0;
        for guild_id in &guild_ids {
            if let Some(guild) = cache.guild(*guild_id) {
                users += guild.member_count;
            }
        }

        let payload = OmniplexStats {
            servers,
            shards,
            users,
            shard_list: (0..shards).collect(),
            status: "online",
        };

        let result = client
            .post("https://api.omniplex.gg/bots/stats")
            .header("Authorization", format!("Bot {}", token))
            .json(&payload)
            .send()
            .await;

        match result {
            Ok(resp) if !resp.status().is_success() => {
                warn!(
                    "Omniplex stats post rejected: {} {}",
                    resp.status(),
                    resp.text().await.unwrap_or_default()
                );
            }
            Err(e) => warn!("Failed to post Omniplex stats: {:?}", e),
            _ => {}
        }
    }
}
