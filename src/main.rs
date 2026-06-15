mod auth;
mod config;
mod db;

use config::Config;

#[tokio::main]
async fn main() {
    let config = Config::from_env();
    println!("Link shortener starting on {}", config.listen_addr);
}