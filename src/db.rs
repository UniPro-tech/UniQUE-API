use sea_orm::Database;
pub type DbConn = sea_orm::DatabaseConnection;

pub async fn connect() -> Result<DbConn, sea_orm::DbErr> {
    let url = std::env::var("DATABASE_URL").expect("DATABASE_URL not set");
    Database::connect(&url).await
}
