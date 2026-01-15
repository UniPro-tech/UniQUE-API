use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;
use utoipa::ToSchema;

use crate::{middleware::auth::AuthUser, models::user, utils::password};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{id}/password/change", put(password_change))
        .route("/users/password/reset", post(password_reset))
}

#[derive(serde::Deserialize, ToSchema)]
pub struct PasswordChange {
    pub current_password: String,
    pub new_password: String,
}

/// ユーザーのパスワードをチェンジするための関数
#[utoipa::path(
    put,
    path = "/users/{id}/password/change",
    tag = "users",
    params(
        ("id" = String, Path, description = "ユーザーID")
    ),
    request_body = PasswordChange,
    responses(
        (status = 201, description = "パスワード変更成功"),
        (status = 401, description = "現在のパスワードが不正確"),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ユーザーが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn password_change(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<PasswordChange>,
) -> Result<impl IntoResponse, StatusCode> {
    // 自分のパスワード変更のみ許可
    if auth_user.user_id != id {
        return Err(StatusCode::FORBIDDEN);
    }

    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        // パスワードの検証
        let password_matches = user
            .password_hash
            .as_deref()
            .map(|h| password::verify_password(&payload.current_password, h))
            .unwrap_or(false);
        if !password_matches {
            return Err(StatusCode::UNAUTHORIZED);
        }

        // 新しいパスワードのハッシュ化
        let new_password_hash = password::hash_password(&payload.new_password);

        let mut am: user::ActiveModel = user.into();
        am.password_hash = Set(Some(new_password_hash));
        am.updated_at = Set(Some(chrono::Utc::now().naive_utc()));
        am.update(&db).await.unwrap();
        return Ok((StatusCode::CREATED, Json(serde_json::Value::Null)));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize, ToSchema)]
pub struct PasswordReset {
    pub username: String,
}

/// ユーザーのパスワードをリセットするための関数
#[utoipa::path(
    post,
    path = "/users/password/reset",
    tag = "users",
    request_body = PasswordReset,
    responses(
        (status = 200, description = "パスワードリセットリンク送信成功"),
        (status = 404, description = "ユーザーが見つからない")
    )
)]
pub async fn password_reset(
    State(db): State<DbConn>,
    Json(payload): Json<PasswordReset>,
) -> impl IntoResponse {
    let found = user::Entity::find()
        .filter(user::Column::CustomId.eq(&payload.username))
        .one(&db)
        .await
        .unwrap();
    if let Some(user) = found {
        // TODO: 外部メールアドレスに対してリセットリンクを送信する処理を追加する
        return (StatusCode::OK, Json(serde_json::Value::Null));
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
