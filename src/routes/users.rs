use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use chrono::Utc;
use sea_orm::*;
use serde_json;

use crate::models::user::{self, Entity as User};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users", get(get_all_users)).route(
        "/users/{id}",
        get(get_user)
            .patch(patch_update_user)
            .delete(delete_user)
            .post(create_user)
            .put(put_user),
    )
    //.merge(users_sub::books::routes())
}

/// すべてのユーザーを取得するための関数
async fn get_all_users(State(db): State<DbConn>) -> Json<serde_json::Value> {
    let users = User::find().all(&db).await.unwrap();
    Json(serde_json::json!({ "data": users }))
}

/// 特定のユーザーを取得するための関数
async fn get_user(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let user = User::find_by_id(id).one(&db).await.unwrap();

    if let Some(user) = user {
        (StatusCode::OK, Json(Some(user)))
    } else {
        (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
    }
}

#[derive(serde::Deserialize)]
struct CreateUser {
    pub custom_id: String,
    pub name: String,
    pub password_hash: Option<String>,
    pub email: Option<String>,
    pub external_email: String,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: bool,
    pub is_enable: Option<bool>,
}

/// 新しいユーザーを作成するための関数
/// システム専用
async fn create_user(
    State(db): State<DbConn>,
    Json(payload): Json<CreateUser>,
) -> impl IntoResponse {
    let am = user::ActiveModel {
        id: Set(uuid::Uuid::new_v4().to_string()),
        custom_id: Set(payload.custom_id),
        name: Set(Some(payload.name)),
        password_hash: Set(payload.password_hash),
        email: Set(payload.email),
        external_email: Set(payload.external_email),
        period: Set(payload.period),
        joined_at: Set(payload.joined_at),
        is_system: Set(payload.is_system),
        created_at: Set(Some(Utc::now().naive_utc())),
        updated_at: Set(Some(Utc::now().naive_utc())),
        is_enable: Set(Some(payload.is_enable.unwrap_or(false))),
    };
    let res = am.insert(&db).await.unwrap();
    (StatusCode::CREATED, Json(res))
}

async fn put_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<CreateUser>,
) -> impl IntoResponse {
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        am.name = Set(Some(payload.name));
        am.external_email = Set(payload.external_email);
        am.password_hash = Set(payload.password_hash);
        am.joined_at = Set(payload.joined_at);
        am.is_system = Set(payload.is_system);
        am.is_enable = Set(Some(payload.is_enable.unwrap_or(false)));
        am.email = Set(payload.email);
        am.period = Set(payload.period);
        am.updated_at = Set(Some(Utc::now().naive_utc()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}

#[derive(serde::Deserialize)]
struct UpdateUser {
    pub name: Option<String>,
    pub password_hash: Option<String>,
    pub external_email: Option<String>,
}

/// ユーザーを差分アップデートするための関数
///
/// > このエンドポイントはOAuthの**アクセストークンでアクセス可能**です。
/// > ただし、システムのユーザーではない場合は、以下のフィールドのみ書き換え可能です。
/// > - name
/// > - external_email
async fn patch_update_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<UpdateUser>,
) -> impl IntoResponse {
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        if let Some(name) = payload.name {
            am.name = Set(Some(name));
        }
        if let Some(external_email) = payload.external_email {
            am.external_email = Set(external_email);
        }
        if let Some(password_hash) = payload.password_hash {
            am.password_hash = Set(Some(password_hash));
        }
        am.updated_at = Set(Some(Utc::now().naive_utc()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}

/// ユーザーを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_user(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let found = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let am: user::ActiveModel = user.into();
        am.delete(&db).await.unwrap();
        return (StatusCode::NO_CONTENT, Json::<Option<user::Model>>(None));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}
