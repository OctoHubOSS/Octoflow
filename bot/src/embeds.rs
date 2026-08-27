//  Copyright (C) 2026 NodeByte LTD

use poise::serenity_prelude::{CreateEmbed, CreateEmbedFooter};

pub const BRAND_COLOR: u32 = 0x6366F1;

pub const SITE_URL: &str = "https://octoflow.ca";
pub const DOCS_URL: &str = "https://octoflow.ca/docs";
pub const STATUS_URL: &str = "https://octoflow.ca/status";
pub const DASHBOARD_URL: &str = "https://octoflow.ca/dashboard";
pub const SUPPORT_URL: &str = "https://discord.gg/Sj2SWMZe2J";


pub fn base() -> CreateEmbed<'static> {
    CreateEmbed::new()
        .color(BRAND_COLOR)
        .footer(CreateEmbedFooter::new(format!("Octoflow | {DOCS_URL}")))
}
