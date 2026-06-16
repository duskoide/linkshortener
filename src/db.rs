use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};

pub async fn init_db(database_url: &str) -> Result<SqlitePool, sqlx::Error> {
    let pool = SqlitePoolOptions::new()
        .connect(database_url)
        .await?;

    // Run migrations
    sqlx::query(include_str!("../migrations/001_initial.sql"))
        .execute(&pool)
        .await?;
    sqlx::query(include_str!("../migrations/002_sessions.sql"))
        .execute(&pool)
        .await?;

    Ok(pool)
}

pub async fn get_password_hash(pool: &SqlitePool) -> Result<Option<String>, sqlx::Error> {
    let result: Option<(String,)> = sqlx::query_as("SELECT value FROM settings WHERE key = 'password_hash'")
        .fetch_optional(pool)
        .await?;
    
    Ok(result.map(|r| r.0))
}

pub async fn set_password_hash(pool: &SqlitePool, hash: &str) -> Result<(), sqlx::Error> {
    sqlx::query("INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)")
        .bind(hash)
        .execute(pool)
        .await?;
    
    Ok(())
}

pub async fn create_session(pool: &SqlitePool, token: &str, expires_in_hours: i64) -> Result<(), sqlx::Error> {
    sqlx::query("INSERT INTO sessions (token, expires_at) VALUES (?, datetime('now', '+' || ? || ' hours'))")
        .bind(token)
        .bind(expires_in_hours)
        .execute(pool)
        .await?;
    
    Ok(())
}

pub async fn validate_session(pool: &SqlitePool, token: &str) -> Result<bool, sqlx::Error> {
    let result: Option<(String,)> = sqlx::query_as("SELECT token FROM sessions WHERE token = ? AND expires_at > datetime('now')")
        .bind(token)
        .fetch_optional(pool)
        .await?;
    
    Ok(result.is_some())
}

pub async fn delete_session(pool: &SqlitePool, token: &str) -> Result<(), sqlx::Error> {
    sqlx::query("DELETE FROM sessions WHERE token = ?")
        .bind(token)
        .execute(pool)
        .await?;
    
    Ok(())
}

pub async fn cleanup_expired_sessions(pool: &SqlitePool) -> Result<(), sqlx::Error> {
    sqlx::query("DELETE FROM sessions WHERE expires_at <= datetime('now')")
        .execute(pool)
        .await?;
    
    Ok(())
}
