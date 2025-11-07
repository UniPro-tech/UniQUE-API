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

use crate::{
    models::user::{self, Entity as User},
    routes::users_sub,
    utils::password,
};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users", get(get_all_users))
        .route(
            "/users/{id}",
            get(get_user)
                .patch(patch_update_user)
                .delete(delete_user)
                .post(create_user)
                .put(put_user),
        )
        .merge(users_sub::discord::routes())
        .merge(users_sub::roles::routes())
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
    pub password: String,
    pub email: String,
    pub external_email: String,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
}

/// 新しいユーザーを作成するための関数
async fn create_user(
    State(db): State<DbConn>,
    Json(payload): Json<CreateUser>,
) -> impl IntoResponse {
    let password_hash = password::hash_password(&payload.password);

    let am = user::ActiveModel {
        id: Set(uuid::Uuid::new_v4().to_string()),
        custom_id: Set(payload.custom_id),
        name: Set(payload.name),
        password_hash: Set(Some(password_hash)),
        email: Set(payload.email),
        external_email: Set(payload.external_email),
        period: Set(payload.period),
        joined_at: Set(payload.joined_at),
        is_system: Set(Some(payload.is_system.unwrap_or(false))),
        created_at: Set(Some(Utc::now().naive_utc())),
        updated_at: Set(Some(Utc::now().naive_utc())),
        is_enable: Set(Some(payload.is_enable.unwrap_or(false))),
        is_suspended: Set(Some(payload.is_suspended.unwrap_or(false))),
        suspended_until: Set(payload.suspended_until),
        suspended_reason: Set(payload.suspended_reason),
    };
    let res = am.insert(&db).await.unwrap();
    (StatusCode::CREATED, Json(res))
}

async fn put_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<CreateUser>,
) -> impl IntoResponse {
    let password_hash = password::hash_password(&payload.password);

    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        am.name = Set(payload.name);
        am.external_email = Set(payload.external_email);
        am.password_hash = Set(password_hash);
        am.joined_at = Set(payload.joined_at);
        am.is_system = Set(Some(payload.is_system.unwrap_or(false)));
        am.is_enable = Set(Some(payload.is_enable.unwrap_or(false)));
        am.email = Set(payload.email);
        am.period = Set(payload.period);
        am.updated_at = Set(Some(Utc::now().naive_utc()));
        am.suspended_until = Set(payload.suspended_until);
        am.suspended_reason = Set(payload.suspended_reason);
        am.is_suspended = Set(Some(payload.is_suspended.unwrap_or(false)));
        am.joined_at = Set(payload.joined_at);
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}

#[derive(serde::Deserialize)]
struct UpdateUser {
    pub custom_id: Option<String>,
    pub name: Option<String>,
    pub password_hash: Option<String>,
    pub external_email: Option<String>,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
    pub email: Option<String>,
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
        if let Some(custom_id) = payload.custom_id {
            am.custom_id = Set(custom_id);
        }
        if let Some(name) = payload.name {
            am.name = Set(name);
        }
        if let Some(external_email) = payload.external_email {
            am.external_email = Set(external_email);
        }
        if let Some(password_hash) = payload.password_hash {
            am.password_hash = Set(Some(password_hash));
        }
        if let Some(period) = payload.period {
            am.period = Set(Some(period));
        }
        if let Some(joined_at) = payload.joined_at {
            am.joined_at = Set(Some(joined_at));
        }
        if let Some(is_system) = payload.is_system {
            am.is_system = Set(Some(is_system));
        }
        if let Some(is_enable) = payload.is_enable {
            am.is_enable = Set(Some(is_enable));
        }
        if let Some(is_suspended) = payload.is_suspended {
            am.is_suspended = Set(Some(is_suspended));
        }
        if let Some(suspended_until) = payload.suspended_until {
            am.suspended_until = Set(Some(suspended_until));
        }
        if let Some(suspended_reason) = payload.suspended_reason {
            am.suspended_reason = Set(Some(suspended_reason));
        }
        if let Some(email) = payload.email {
            am.email = Set(email);
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
