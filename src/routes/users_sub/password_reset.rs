use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;

use crate::models::user;
use crate::utils::password;
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users/{id}/password_reset", put(password_reset))
}

#[derive(serde::Deserialize)]
struct PasswordReset {
    pub current_password: String,
    pub new_password: String,
}

/// ユーザーのパスワードをリセットするための関数
async fn password_reset(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<PasswordReset>,
) -> impl IntoResponse {
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        // パスワードの検証
        let password_matches = user
            .password_hash
            .as_deref()
            .map(|h| password::verify_password(&payload.current_password, h))
            .unwrap_or(false);
        if !password_matches {
            return (
                StatusCode::UNAUTHORIZED,
                Json(serde_json::json!({ "error": "Invalid current password" })),
            );
        }

        // 新しいパスワードのハッシュ化
        let new_password_hash = password::hash_password(&payload.new_password);

        let am = user::ActiveModel {
            id: Set(user.id),
            password_hash: Set(Some(new_password_hash)),
            ..Default::default()
        };
        am.insert(&db).await.unwrap();
        return (StatusCode::CREATED, Json(serde_json::Value::Null));
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
