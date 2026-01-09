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
use ulid::Ulid;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::role::{self, Entity as Role},
    routes::roles_sub,
};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/roles", get(get_all_roles))
        .route(
            "/roles/{id}",
            get(get_role)
                .patch(patch_update_role)
                .delete(delete_role)
                .post(create_role)
                .put(put_role),
        )
        .merge(roles_sub::search::routes())
}

/// すべてのロールを取得するための関数
async fn get_all_roles(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let roles = Role::find().all(&db).await.unwrap();
    Ok(Json(serde_json::json!({ "data": roles })))
}

/// 特定のロールを取得するための関数
async fn get_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let role = Role::find_by_id(id).one(&db).await.unwrap();

    if let Some(role) = role {
        Ok((StatusCode::OK, Json(Some(role))))
    } else {
        Err(StatusCode::NOT_FOUND)
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
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateRole>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let am = role::ActiveModel {
        id: Set(Ulid::new().to_string()),
        custom_id: Set(payload.custom_id),
        name: Set(Some(payload.name)),
        permission: Set(payload.permission),
        created_at: Set(Utc::now()),
        updated_at: Set(Utc::now()),
        is_enable: Set(Some(payload.is_enable.unwrap_or(true))),
        is_system: Set(Some(payload.is_system.unwrap_or(false))),
    };
    let res = am.insert(&db).await.unwrap();
    Ok((StatusCode::CREATED, Json(res)))
}

async fn put_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateRole>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let found = role::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: role::ActiveModel = user.into();
        am.custom_id = Set(payload.custom_id);
        am.name = Set(Some(payload.name));
        am.permission = Set(payload.permission);
        am.is_system = Set(Some(payload.is_system.unwrap_or(false)));
        am.is_enable = Set(Some(payload.is_enable.unwrap_or(false)));
        am.updated_at = Set(Utc::now());
        let res = am.update(&db).await.unwrap();
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize)]
struct UpdateRole {
    pub name: Option<String>,
    pub custom_id: Option<String>,
    pub permission: Option<i32>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
}

/// ロールを差分アップデートするための関数
async fn patch_update_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<UpdateRole>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let found = role::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: role::ActiveModel = user.into();
        if let Some(name) = payload.name {
            am.name = Set(Some(name));
        }
        if let Some(custom_id) = payload.custom_id {
            am.custom_id = Set(custom_id);
        }
        if let Some(permission) = payload.permission {
            am.permission = Set(permission);
        }
        if let Some(is_system) = payload.is_system {
            am.is_system = Set(Some(is_system));
        }
        if let Some(is_enable) = payload.is_enable {
            am.is_enable = Set(Some(is_enable));
        }
        am.updated_at = Set(Utc::now());
        let res = am.update(&db).await.unwrap();
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

/// ロールを削除するための関数
async fn delete_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let found = Role::find_by_id(id).one(&db).await.unwrap();
    if let Some(role) = found {
        let am: role::ActiveModel = role.into();
        am.delete(&db).await.unwrap();
        return Ok((StatusCode::NO_CONTENT, Json::<Option<role::Model>>(None)));
    }
    Err(StatusCode::NOT_FOUND)
}
