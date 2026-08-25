use crate::Context;
use crate::Error;

#[poise::command(category = "Help", track_edits, slash_command)]
pub async fn help(ctx: Context<'_>, command: Option<String>) -> Result<(), Error> {
    // No prefix commands remain (slash-only now), but botox::help::help's
    // subcommand line template always interpolates a prefix string. Passing
    // "" is the closest we can get without forking botox — leaves a doubled
    // command name with no separator on subcommand lines instead of a
    // misleading "git!" that no longer does anything.
    botox::help::help(ctx, command, "", botox::help::HelpOptions::default()).await
}

#[poise::command(category = "Help", slash_command, user_cooldown = 1)]
pub async fn simplehelp(
    ctx: Context<'_>,
    #[description = "Specific command to show help about"]
    #[autocomplete = "poise::builtins::autocomplete_command"]
    command: Option<String>,
) -> Result<(), Error> {
    botox::help::simplehelp(ctx, command).await
}
