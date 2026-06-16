use axum::{
    extract::Extension,
    routing::{get, post, delete},
    Router,
};
use std::net::SocketAddr;
use tera::Tera;

mod auth;
mod config;
mod db;
mod routes;

use routes::AppState;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();

    let config = config::Config::from_env();

    let pool = db::init_db(&config.database_url)
        .await
        .expect("Failed to initialize database");

    let tera = Tera::new("templates/**/*").expect("Failed to initialize Tera templates");

    let state = AppState {
        pool: pool.clone(),
        tera,
    };

    let app = Router::new()
        .route("/", get(routes::dashboard))
        .route("/login", get(routes::login_page).post(routes::login_post))
        .route("/logout", post(routes::logout))
        .route("/dashboard", get(routes::dashboard))
        .route("/create", get(routes::create_page).post(routes::create_post))
        .route("/api/links", get(routes::api_links))
        .route("/api/links/:id", delete(routes::api_delete_link))
        .route("/:short_code", get(routes::redirect_handler))
        .with_state(state)
        .layer(Extension(pool));

    let addr: SocketAddr = config.listen_addr.parse().expect("Invalid listen address");
    println!("Link shortener listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
