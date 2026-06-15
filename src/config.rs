use std::env;

#[derive(Clone)]
pub struct Config {
    pub database_url: String,
    pub password_hash: Option<String>,
    pub listen_addr: String,
    pub session_secret: String,
}

impl Config {
    pub fn from_env() -> Self {
        dotenvy::dotenv().ok();

        Self {
            database_url: env::var("DATABASE_URL").unwrap_or_else(|_| "sqlite:data/links.db".to_string()),
            password_hash: env::var("PASSWORD_HASH").ok().filter(|s| !s.is_empty()),
            listen_addr: env::var("LISTEN_ADDR").unwrap_or_else(|_| "0.0.0.0:8081".to_string()),
            session_secret: env::var("SESSION_SECRET").unwrap_or_else(|_| "default-secret-change-me".to_string()),
        }
    }
}