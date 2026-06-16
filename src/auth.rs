use axum::{
    extract::FromRequestParts,
    http::{request::Parts, StatusCode},
    response::Redirect,
};
use axum_extra::extract::cookie::CookieJar;
use bcrypt::{hash, verify, DEFAULT_COST};
use rand::Rng;
use sqlx::SqlitePool;

#[derive(Clone)]
pub struct AuthSession {
    pub is_authenticated: bool,
}

impl AuthSession {
    pub fn new() -> Self {
        Self {
            is_authenticated: false,
        }
    }
}

pub async fn hash_password(password: &str) -> Result<String, bcrypt::BcryptError> {
    hash(password, DEFAULT_COST)
}

pub async fn verify_password(password: &str, hash: &str) -> Result<bool, bcrypt::BcryptError> {
    verify(password, hash)
}

pub fn generate_session_token() -> String {
    let mut rng = rand::thread_rng();
    let token: String = (0..32)
        .map(|_| {
            let idx = rng.gen_range(0..62);
            match idx {
                0..=9 => (b'0' + idx) as char,
                10..=35 => (b'a' + idx - 10) as char,
                36..=61 => (b'A' + idx - 36) as char,
                _ => unreachable!(),
            }
        })
        .collect();
    token
}

#[axum::async_trait]
impl<S> FromRequestParts<S> for AuthSession
where
    S: Send + Sync,
{
    type Rejection = (StatusCode, Redirect);

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let jar = CookieJar::from_request_parts(parts, state).await
            .map_err(|_| (StatusCode::INTERNAL_SERVER_ERROR, Redirect::to("/login")))?;

        let session_token = jar.get("session").map(|c| c.value().to_string());

        if let Some(token) = session_token {
            // Try to extract pool from state using trait bounds
            // Since we can't directly access AppState here, we'll use a different approach
            // We need to use axum's state extraction
            let pool = parts.extensions.get::<SqlitePool>()
                .ok_or((StatusCode::INTERNAL_SERVER_ERROR, Redirect::to("/login")))?;
            
            // Validate token against database
            let is_valid = crate::db::validate_session(pool, &token).await
                .map_err(|_| (StatusCode::INTERNAL_SERVER_ERROR, Redirect::to("/login")))?;
            
            if is_valid {
                Ok(AuthSession {
                    is_authenticated: true,
                })
            } else {
                Err((StatusCode::UNAUTHORIZED, Redirect::to("/login")))
            }
        } else {
            Err((StatusCode::UNAUTHORIZED, Redirect::to("/login")))
        }
    }
}
