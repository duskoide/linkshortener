use axum::{
    extract::FromRequestParts,
    http::{request::Parts, StatusCode},
    response::{IntoResponse, Redirect, Response},
};
use axum_extra::extract::cookie::CookieJar;
use bcrypt::{hash, verify, DEFAULT_COST};
use rand::Rng;

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

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let jar = CookieJar::from_request_parts(parts, _state).await.unwrap();

        let session_token = jar.get("session").map(|c| c.value().to_string());

        if let Some(_token) = session_token {
            Ok(AuthSession {
                is_authenticated: true,
            })
        } else {
            Err((StatusCode::UNAUTHORIZED, Redirect::to("/login")))
        }
    }
}
