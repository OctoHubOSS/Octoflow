use once_cell::sync::Lazy;
use serde::{Deserialize, Serialize};
use std::{fs::File, io::Write};

use crate::Error;

pub static CONFIG: Lazy<Config> = Lazy::new(|| Config::load().expect("Failed to load config"));

#[derive(Serialize, Deserialize)]
pub struct Config {
    pub database_url: String,
    pub token: String,
    pub api_url: Vec<String>,
    pub proxy_url: Option<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            database_url: String::from(""),
            token: String::from(""),
            api_url: vec![String::from("https://v2.gitlogs.xyz")],
            proxy_url: Some(String::from("http://127.0.0.1:3219")),
        }
    }
}

impl Config {
    pub fn load() -> Result<Self, Error> {
        if std::path::Path::new("config.yaml.sample").exists() {
            std::fs::remove_file("config.yaml.sample")?;
        }

        let mut sample = File::create("config.yaml.sample")?;

        sample.write_all(serde_yaml::to_string(&Config::default())?.as_bytes())?;

        let file = File::open("config.yaml");

        match file {
            Ok(file) => {
                let cfg: Config = serde_yaml::from_reader(file)?;

                if cfg.api_url.is_empty() {
                    panic!("At least one api URL must be provided")
                }
                Ok(cfg)
            }
            Err(e) => {
                println!("config.yaml could not be loaded: {}", e);
                std::process::exit(1);
            }
        }
    }
}
