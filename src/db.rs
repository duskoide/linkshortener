use sqlx::sqlite::{SqlitePool, SqlitePoolOptions};

pub async fn init_db(database_url: &str) -> Result<SqlitePool, sqlx::Error> {
    let pool = SqlitePoolOptions::new()
        .connect(database_url)
        .await?;

    // Run migrations
    sqlx::query(include_str!("../migrations/001_initial.sql"))
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
