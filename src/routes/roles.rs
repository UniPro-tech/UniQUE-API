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

use crate::models::role::{self, Entity as Role};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/roles", get(get_all_roles)).route(
        "/roles/{id}",
        get(get_role)
            .patch(patch_update_role)
            .delete(delete_role)
            .post(create_role)
            .put(put_role),
    )
    //.merge(users_sub::books::routes())
}

/// すべてのロールを取得するための関数
async fn get_all_roles(State(db): State<DbConn>) -> Json<serde_json::Value> {
    let roles = Role::find().all(&db).await.unwrap();
    Json(serde_json::json!({ "data": roles }))
}

/// 特定のロールを取得するための関数
async fn get_role(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let role = Role::find_by_id(id).one(&db).await.unwrap();

    if let Some(role) = role {
        (StatusCode::OK, Json(Some(role)))
    } else {
        (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None))
    }
}

#[derive(serde::Deserialize)]
struct CreateRole {
    pub custom_id: String,
    pub name: String,
    pub permission: i32,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
}

/// 新しいロールを作成するための関数
/// システム専用
async fn create_role(
    State(db): State<DbConn>,
    Json(payload): Json<CreateRole>,
) -> impl IntoResponse {
    let am = role::ActiveModel {
        id: Set(uuid::Uuid::new_v4().to_string()),
        custom_id: Set(Some(payload.custom_id)),
        name: Set(Some(payload.name)),
        permission: Set(payload.permission),
        created_at: Set(Utc::now()),
        updated_at: Set(Some(Utc::now())),
        is_enable: Set(payload.is_enable.unwrap_or(true)),
        is_system: Set(payload.is_system.unwrap_or(false)),
    };
    let res = am.insert(&db).await.unwrap();
    (StatusCode::CREATED, Json(res))
}

async fn put_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<CreateRole>,
) -> impl IntoResponse {
    let found = role::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: role::ActiveModel = user.into();
        am.custom_id = Set(Some(payload.custom_id));
        am.name = Set(Some(payload.name));
        am.permission = Set(payload.permission);
        am.is_system = Set(payload.is_system.unwrap_or(false));
        am.is_enable = Set(payload.is_enable.unwrap_or(false));
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None))
}

#[derive(serde::Deserialize)]
struct UpdateRole {
    pub name: Option<String>,
    pub password_hash: Option<String>,
    pub external_email: Option<String>,
    pub custom_id: Option<String>,
    pub permission: Option<i32>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
}

/// ユーザーを差分アップデートするための関数
///
/// > このエンドポイントはOAuthの**アクセストークンでアクセス可能**です。
/// > ただし、システムのユーザーではない場合は、以下のフィールドのみ書き換え可能です。
/// > - name
/// > - external_email
async fn patch_update_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<UpdateRole>,
) -> impl IntoResponse {
    let found = role::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: role::ActiveModel = user.into();
        if let Some(name) = payload.name {
            am.name = Set(Some(name));
        }
        if let Some(custom_id) = payload.custom_id {
            am.custom_id = Set(Some(custom_id));
        }
        if let Some(permission) = payload.permission {
            am.permission = Set(permission);
        }
        if let Some(is_system) = payload.is_system {
            am.is_system = Set(is_system);
        }
        if let Some(is_enable) = payload.is_enable {
            am.is_enable = Set(is_enable);
        }
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None))
}

/// ユーザーを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_role(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let found = Role::find_by_id(id).one(&db).await.unwrap();
    if let Some(role) = found {
        let am: role::ActiveModel = role.into();
        am.delete(&db).await.unwrap();
        return (StatusCode::NO_CONTENT, Json::<Option<role::Model>>(None));
    }
    (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None))
}
