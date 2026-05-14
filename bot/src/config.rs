use once_cell::sync::Lazy;
use serde::{Deserialize, Serialize};
use std::{fs::File, io::Write};

use crate::Error;

/// Global config object
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
            proxy_url: None,
        }
    }
}

impl Config {
    pub fn load() -> Result<Self, Error> {
        // Delete config.yaml.sample if it exists
        if std::path::Path::new("config.yaml.sample").exists() {
            std::fs::remove_file("config.yaml.sample")?;
        }

        // Create config.yaml.sample
        let mut sample = File::create("config.yaml.sample")?;

        // Write default config to config.yaml.sample
        sample.write_all(serde_yaml::to_string(&Config::default())?.as_bytes())?;

        // Open config.yaml
        let file = File::open("config.yaml");

        match file {
            Ok(file) => {
                // Parse config.yaml
                let cfg: Config = serde_yaml::from_reader(file)?;

                if cfg.api_url.is_empty() {
                    panic!("At least one api URL must be provided")
                }

                // Return config
                Ok(cfg)
            }
            Err(e) => {
                // Print error
                println!("config.yaml could not be loaded: {}", e);

                // Exit
                std::process::exit(1);
            }
        }
    }
}
