use crate::Context;
use crate::Error;

#[poise::command(category = "Help", track_edits, prefix_command, slash_command)]
pub async fn help(ctx: Context<'_>, command: Option<String>) -> Result<(), Error> {
    botox::help::help(ctx, command, "git!", botox::help::HelpOptions::default()).await
}

#[poise::command(category = "Help", prefix_command, slash_command, user_cooldown = 1)]
pub async fn simplehelp(
    ctx: Context<'_>,
    #[description = "Specific command to show help about"]
    command: Option<String>,
) -> Result<(), Error> {
    botox::help::simplehelp(ctx, command).await
}
