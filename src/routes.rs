use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::{Html, Redirect},
    Form,
};
use axum_extra::extract::cookie::{Cookie, CookieJar};
use serde::{Deserialize, Serialize};
use serde_json;
use sqlx::{FromRow, SqlitePool};
use tera::Tera;

use crate::auth::{hash_password, verify_password, AuthSession, generate_session_token};
#[derive(Clone)]
pub struct AppState {
    pub pool: SqlitePool,
    pub tera: Tera,
}

#[derive(Deserialize)]
pub struct LoginForm {
    password: String,
}

#[derive(Deserialize)]
pub struct CreateLinkForm {
    url: String,
    custom_code: Option<String>,
}

#[derive(Serialize, FromRow)]
pub struct Link {
    pub id: i64,
    pub short_code: String,
    pub original_url: String,
    pub created_at: String,
    pub click_count: i64,
}

pub async fn login_page(State(state): State<AppState>) -> Html<String> {
    let context = tera::Context::new();
    Html(state.tera.render("login.html", &context).unwrap())
}

pub async fn login_post(
    State(state): State<AppState>,
    jar: CookieJar,
    Form(form): Form<LoginForm>,
) -> Result<(CookieJar, Redirect), (StatusCode, Html<String>)> {
    let stored_hash = crate::db::get_password_hash(&state.pool)
        .await
        .unwrap_or(None);

    let hash = match stored_hash {
        Some(h) => h,
        None => {
            let new_hash = hash_password(&form.password).await.unwrap();
            crate::db::set_password_hash(&state.pool, &new_hash).await.unwrap();
            new_hash
        }
    };

    if verify_password(&form.password, &hash).await.unwrap() {
        let session_token = generate_session_token();
        
        // Store session in database (expires in 24 hours)
        if let Err(e) = crate::db::create_session(&state.pool, &session_token, 24).await {
            tracing::error!("Failed to create session: {}", e);
            return Err((
                StatusCode::INTERNAL_SERVER_ERROR,
                Html(state.tera.render("error.html", &tera::Context::new()).unwrap_or_else(|_| "<h1>Internal Server Error</h1>".to_string())),
            ));
        }

        let cookie = Cookie::build(("session", session_token))
            .path("/")
            .http_only(true)
            .build();

        Ok((jar.add(cookie), Redirect::to("/dashboard")))
    } else {
        let mut context = tera::Context::new();
        context.insert("error", "Invalid password");
        Err((StatusCode::UNAUTHORIZED, Html(state.tera.render("login.html", &context).unwrap())))
    }
}

pub async fn logout(
    State(state): State<AppState>,
    jar: CookieJar,
) -> Result<(CookieJar, Redirect), (StatusCode, Html<String>)> {
    // Get the session token from cookie
    let session_token = jar.get("session").map(|c| c.value().to_string());
    
    if let Some(token) = session_token {
        // Delete session from database
        if let Err(e) = crate::db::delete_session(&state.pool, &token).await {
            tracing::error!("Failed to delete session: {}", e);
        }
    }

    // Remove the cookie
    let cookie = Cookie::build("session")
        .path("/")
        .http_only(true)
        .max_age(time::Duration::ZERO)
        .build();

    Ok((jar.remove(cookie), Redirect::to("/login")))
}

pub async fn dashboard(
    _auth: AuthSession,
    State(state): State<AppState>,
) -> Result<Html<String>, (StatusCode, Html<String>)> {
    let links: Vec<Link> = sqlx::query_as::<_, Link>("SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC")
        .fetch_all(&state.pool)
        .await
        .map_err(|e| {
            tracing::error!("Failed to fetch links: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Html(state.tera.render("error.html", &tera::Context::new()).unwrap_or_else(|_| "<h1>Internal Server Error</h1>".to_string())),
            )
        })?;

    let mut context = tera::Context::new();
    context.insert("links", &links);
    Ok(Html(state.tera.render("dashboard.html", &context).unwrap()))
}

pub async fn create_page(
    _auth: AuthSession,
    State(state): State<AppState>,
) -> Html<String> {
    let context = tera::Context::new();
    Html(state.tera.render("create.html", &context).unwrap())
}

pub async fn create_post(
    _auth: AuthSession,
    State(state): State<AppState>,
    Form(form): Form<CreateLinkForm>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    let short_code = form.custom_code.unwrap_or_else(|| generate_short_code());

    if short_code.len() < 3 || short_code.len() > 20 || !short_code.chars().all(|c| c.is_alphanumeric()) {
        let mut context = tera::Context::new();
        context.insert("error", "Short code must be 3-20 alphanumeric characters");
        return Err((StatusCode::BAD_REQUEST, Html(state.tera.render("create.html", &context).unwrap())));
    }

    if !form.url.starts_with("http://") && !form.url.starts_with("https://") {
        let mut context = tera::Context::new();
        context.insert("error", "URL must start with http:// or https://");
        return Err((StatusCode::BAD_REQUEST, Html(state.tera.render("create.html", &context).unwrap())));
    }

    let result = sqlx::query("INSERT INTO links (short_code, original_url) VALUES (?, ?)")
        .bind(&short_code)
        .bind(&form.url)
        .execute(&state.pool)
        .await;

    match result {
        Ok(_) => Ok(Redirect::to("/dashboard")),
        Err(sqlx::Error::Database(e)) if e.message().contains("UNIQUE constraint") => {
            let mut context = tera::Context::new();
            context.insert("error", "Short code already exists");
            Err((StatusCode::CONFLICT, Html(state.tera.render("create.html", &context).unwrap())))
        }
        Err(_) => {
            let mut context = tera::Context::new();
            context.insert("error", "Failed to create link");
            Err((StatusCode::INTERNAL_SERVER_ERROR, Html(state.tera.render("create.html", &context).unwrap())))
        }
    }
}

pub async fn redirect_handler(
    State(state): State<AppState>,
    Path(short_code): Path<String>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    let link: Option<Link> = sqlx::query_as::<_, Link>("SELECT id, short_code, original_url, created_at, click_count FROM links WHERE short_code = ?")
        .bind(&short_code)
        .fetch_optional(&state.pool)
        .await
        .unwrap_or(None);

    match link {
        Some(link) => {
            sqlx::query("UPDATE links SET click_count = click_count + 1 WHERE id = ?")
                .bind(link.id)
                .execute(&state.pool)
                .await
                .ok();

            Ok(Redirect::to(&link.original_url))
        }
        None => {
            let context = tera::Context::new();
            Err((StatusCode::NOT_FOUND, Html(state.tera.render("404.html", &context).unwrap())))
        }
    }
}

pub async fn api_links(
    _auth: AuthSession,
    State(state): State<AppState>,
) -> Result<axum::Json<Vec<Link>>, (StatusCode, axum::Json<serde_json::Value>)> {
    let links: Vec<Link> = sqlx::query_as::<_, Link>("SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC")
        .fetch_all(&state.pool)
        .await
        .map_err(|e| {
            tracing::error!("Failed to fetch links: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                axum::Json(serde_json::json!({"error": "Failed to fetch links"})),
            )
        })?;

    Ok(axum::Json(links))
}

pub async fn api_delete_link(
    _auth: AuthSession,
    State(state): State<AppState>,
    Path(id): Path<i64>,
) -> Result<StatusCode, StatusCode> {
    let result = sqlx::query("DELETE FROM links WHERE id = ?")
        .bind(id)
        .execute(&state.pool)
        .await;

    match result {
        Ok(_) => Ok(StatusCode::OK),
        Err(_) => Err(StatusCode::INTERNAL_SERVER_ERROR),
    }
}

fn generate_short_code() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    (0..6)
        .map(|_| {
            let idx = rng.gen_range(0..62);
            match idx {
                0..=9 => (b'0' + idx) as char,
                10..=35 => (b'a' + idx - 10) as char,
                36..=61 => (b'A' + idx - 36) as char,
                _ => unreachable!(),
            }
        })
        .collect()
}
