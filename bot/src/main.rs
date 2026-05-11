use std::time::Duration;

use log::{error, info};
use poise::serenity_prelude::{
    self as prelude, FullEvent,
};
use sqlx::postgres::PgPoolOptions;
use serenity::gateway::ActivityData;
use std::sync::Arc;

mod help;
mod commands;
mod backups;
mod config;
mod eventmods;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");

type Error = Box<dyn std::error::Error + Send + Sync>;
type Context<'a> = poise::Context<'a, Data, Error>;
// User data, which is stored and accessible in all command invocations
pub struct Data {
    pub pool: sqlx::PgPool,
}

#[poise::command(prefix_command, hide_in_help)]
async fn register(ctx: Context<'_>) -> Result<(), Error> {
    poise::builtins::register_application_commands_buttons(ctx).await?;
    Ok(())
}

async fn on_error(error: poise::FrameworkError<'_, Data, Error>) {
    // This is our custom error handler
    // They are many errors that can occur, so we only handle the ones we want to customize
    // and forward the rest to the default handler
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
            // Always wait a bit here for cache to finish up
            tokio::time::sleep(Duration::from_secs(2)).await;

            info!(
                "{} is ready!",
                data_about_bot.user.name
            );

            // Set activity
            ctx.serenity_context.set_activity(Some(ActivityData::playing("git!help")));
        }
        _ => {}
    }

    Ok(())
}

#[tokio::main]
async fn main() {
    const MAX_CONNECTIONS: u32 = 3; // max connections to the database, we don't need too many here

    std::env::set_var("RUST_LOG", "bot=info");

    env_logger::init();

    let mut http =
        prelude::HttpBuilder::new(&config::CONFIG.token);

    if let Some(v) = &config::CONFIG.proxy_url {
        info!("Setting proxy url to {}", v);
        http = http.proxy(v).ratelimiter_disabled(true);
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

    let framework = poise::Framework::new(
        poise::FrameworkOptions {
            initialize_owners: true,
            prefix_options: poise::PrefixFrameworkOptions {
                prefix: Some("git!".into()),
                ..poise::PrefixFrameworkOptions::default()
            },
            event_handler: |ctx, event| Box::pin(event_listener(ctx, event)),
            commands: vec![
                register(),
                help::simplehelp(),
                help::help(),
                commands::list(),
                commands::newhook(),
                commands::edithook(),
                commands::newrepo(),
                commands::editrepo(),
                commands::delhook(),
                commands::delrepo(),
                commands::setrepochannel(),
                commands::resetsecret(),
                backups::backup(),
                backups::restore(),
                eventmods::eventmod(),
            ],
            // This code is run before every command
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
            // This code is run after every command returns Ok
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

    if let Err(why) = client.start().await {
        error!("Client error: {:?}", why);
    }
}
