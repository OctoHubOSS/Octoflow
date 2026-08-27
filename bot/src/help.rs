use std::fmt::Write as _;
use std::time::Duration;

use futures_util::StreamExt;
use poise::serenity_prelude::{
    ButtonStyle, ComponentInteractionDataKind, CreateActionRow, CreateButton, CreateEmbed,
    CreateSelectMenu, CreateSelectMenuKind, CreateSelectMenuOption, EditInteractionResponse,
};
use poise::CreateReply;

use crate::embeds;
use crate::Context;
use crate::Error;

fn quick_links_embed() -> CreateEmbed<'static> {
    embeds::base()
        .title("Quick links")
        .field("Documentation", format!("[octoflow.ca/docs]({})", embeds::DOCS_URL), true)
        .field("Status", format!("[octoflow.ca/status]({})", embeds::STATUS_URL), true)
        .field("Dashboard", format!("[octoflow.ca/dashboard]({})", embeds::DASHBOARD_URL), true)
        .field("Support server", format!("[Join us]({})", embeds::SUPPORT_URL), true)
}

struct HelpPage {
    category: String,
    body: String,
}

async fn build_pages(ctx: Context<'_>) -> Vec<HelpPage> {
    let mut categories: Vec<(Option<String>, Vec<&poise::Command<crate::Data, Error>>)> = Vec::new();

    for cmd in &ctx.framework().options().commands {
        if let Some(entry) = categories.iter_mut().find(|(c, _)| c == &cmd.category) {
            entry.1.push(cmd);
        } else {
            categories.push((cmd.category.clone(), vec![cmd]));
        }
    }

    let mut pages = Vec::new();

    for (category, commands) in categories {
        let name = category.unwrap_or_else(|| "Uncategorized".to_string());
        let mut body = String::new();

        for command in commands {
            if command.hide_in_help {
                continue;
            }

            let mut allowed = true;
            for check in command.checks.iter() {
                match check(ctx).await {
                    Ok(true) => {}
                    Ok(false) => {
                        allowed = false;
                        break;
                    }
                    Err(_) => continue,
                }
            }
            if !allowed {
                continue;
            }

            let _ = writeln!(
                body,
                "/{} - {}",
                command.name,
                command.description.as_deref().unwrap_or("No description available yet")
            );

            if !command.subcommands.is_empty() {
                for sub in &command.subcommands {
                    if sub.hide_in_help {
                        continue;
                    }
                    let _ = writeln!(
                        body,
                        "  /{} {} - {}",
                        command.name,
                        sub.name,
                        sub.description.as_deref().unwrap_or("No description available yet")
                    );
                }
            }
        }

        if !body.is_empty() {
            pages.push(HelpPage { category: name, body });
        }
    }

    pages
}

fn render_page<'a>(pages: &[HelpPage], index: usize) -> CreateReply<'a> {
    let page = &pages[index];
    let prev_disabled = index == 0;
    let next_disabled = index >= pages.len() - 1;

    let options: Vec<CreateSelectMenuOption> = pages
        .iter()
        .enumerate()
        .map(|(i, p)| {
            let label = if i == index { format!("{} (current)", p.category) } else { p.category.clone() };
            CreateSelectMenuOption::new(label, i.to_string())
        })
        .collect();

    CreateReply::default()
        .embed(
            embeds::base()
                .title(format!("{} (page {} of {})", page.category, index + 1, pages.len()))
                .description(page.body.clone()),
        )
        .components(vec![
            CreateActionRow::Buttons(vec![
                CreateButton::new(format!("help:{}", index.saturating_sub(1)))
                    .label("Previous")
                    .disabled(prev_disabled),
                CreateButton::new("help:cancel").label("Close").style(ButtonStyle::Danger),
                CreateButton::new(format!("help:{}", index + 1))
                    .label("Next")
                    .disabled(next_disabled),
            ]),
            CreateActionRow::SelectMenu(
                CreateSelectMenu::new("help:jump", CreateSelectMenuKind::String { options: options.into() })
                    .custom_id("help:jump"),
            ),
        ])
}

#[poise::command(category = "Help", slash_command)]
pub async fn help(ctx: Context<'_>, #[description = "Specific command to show help about"] command: Option<String>) -> Result<(), Error> {
    if let Some(cmd_name) = command {
        for cmd in &ctx.framework().options().commands {
            if cmd.name != cmd_name {
                continue;
            }

            let params = cmd
                .parameters
                .iter()
                .map(|p| format!("{} - {}", p.name, p.description.as_deref().unwrap_or("No description available yet")))
                .collect::<Vec<_>>()
                .join("\n");

            let mut embed = embeds::base()
                .title(format!("Help: /{}", cmd.name))
                .description(cmd.description.as_deref().unwrap_or("No description available yet").to_string());

            if !params.is_empty() {
                embed = embed.field("Parameters", params, false);
            }

            for sub in &cmd.subcommands {
                let sub_params = sub
                    .parameters
                    .iter()
                    .map(|p| format!("{} - {}", p.name, p.description.as_deref().unwrap_or("No description available yet")))
                    .collect::<Vec<_>>()
                    .join("\n");

                embed = embed.field(
                    format!("/{} {}", cmd.name, sub.name),
                    format!("{}\n{}", sub.description.as_deref().unwrap_or("No description available yet"), sub_params),
                    false,
                );
            }

            ctx.send(CreateReply::default().embed(embed)).await?;
            return Ok(());
        }

        ctx.say(format!("No command named `{}` was found. Run `/help` with no arguments to see everything.", cmd_name)).await?;
        return Ok(());
    }

    let pages = build_pages(ctx).await;

    if pages.is_empty() {
        ctx.say("No commands available.").await?;
        return Ok(());
    }

    let reply = ctx.send(render_page(&pages, 0)).await?;
    ctx.send(CreateReply::default().embed(quick_links_embed())).await?;

    let msg = reply.into_message().await?;

    let mut collector = msg
        .await_component_interactions(ctx.serenity_context().shard.clone())
        .author_id(ctx.author().id)
        .timeout(Duration::from_secs(120))
        .stream();

    while let Some(interaction) = collector.next().await {
        interaction.defer(&ctx.serenity_context().http).await?;

        let id = interaction.data.custom_id.clone();

        if id == "help:cancel" {
            interaction.delete_response(&ctx.serenity_context().http).await?;
            return Ok(());
        }

        let target = if id == "help:jump" {
            match &interaction.data.kind {
                ComponentInteractionDataKind::StringSelect { values } => values.first().and_then(|v| v.parse::<usize>().ok()),
                _ => None,
            }
        } else if let Some(rest) = id.strip_prefix("help:") {
            rest.parse::<usize>().ok()
        } else {
            None
        };

        let Some(target) = target else { continue };
        if target >= pages.len() {
            continue;
        }

        interaction
            .edit_response(
                &ctx.serenity_context().http,
                render_page(&pages, target).to_slash_initial_response_edit(EditInteractionResponse::new()),
            )
            .await?;
    }

    Ok(())
}

#[poise::command(category = "Help", slash_command, user_cooldown = 1)]
pub async fn simplehelp(
    ctx: Context<'_>,
    #[description = "Specific command to show help about"]
    #[autocomplete = "poise::builtins::autocomplete_command"]
    command: Option<String>,
) -> Result<(), Error> {
    let is_overview = command.is_none();

    poise::builtins::help(
        ctx,
        command.as_deref(),
        poise::builtins::HelpConfiguration {
            show_context_menu_commands: true,
            ..poise::builtins::HelpConfiguration::default()
        },
    )
    .await?;

    if is_overview {
        ctx.send(CreateReply::default().embed(quick_links_embed())).await?;
    }

    Ok(())
}
