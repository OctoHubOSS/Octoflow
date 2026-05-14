#[derive(Clone, Copy, Debug, poise::ChoiceParameter)]
pub enum WebhookProvider {
    #[name = "GitHub"]
    #[name = "github"]
    Github,
    #[name = "GitLab"]
    #[name = "gitlab"]
    Gitlab,
}

impl WebhookProvider {
    pub fn as_db(self) -> &'static str {
        match self {
            Self::Github => "github",
            Self::Gitlab => "gitlab",
        }
    }
}
