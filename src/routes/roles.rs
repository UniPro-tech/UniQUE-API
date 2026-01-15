use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use chrono::Utc;
use sea_orm::*;
use serde::Serialize;
use ulid::Ulid;
use utoipa::ToSchema;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::role::{self, Entity as Role},
    routes::{common_dtos::array_dto::ApiResponse, roles_sub},
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct RoleResponse {
    pub id: String,
    pub custom_id: String,
    pub name: Option<String>,
    pub permission: i32,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub created_at: chrono::DateTime<Utc>,
    pub updated_at: chrono::DateTime<Utc>,
}

impl From<role::Model> for RoleResponse {
    fn from(role: role::Model) -> Self {
        Self {
            id: role.id,
            custom_id: role.custom_id,
            name: role.name,
            permission: role.permission,
            is_system: role.is_system,
            is_enable: role.is_enable,
            created_at: role.created_at,
            updated_at: role.updated_at,
        }
    }
}

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
        .merge(roles_sub::permissions::routes())
        .merge(roles_sub::users::routes())
}

/// すべてのロールを取得するための関数
#[utoipa::path(
    get,
    path = "/roles",
    tag = "roles",
    responses(
        (status = 200, description = "ロール一覧の取得に成功", body = ApiResponse<Vec<RoleResponse>>),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_all_roles(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let roles = Role::find().all(&db).await.unwrap();
    let responses: Vec<RoleResponse> = roles.into_iter().map(RoleResponse::from).collect();
    Ok((StatusCode::OK, Json(ApiResponse { data: responses })))
}

/// 特定のロールを取得するための関数
#[utoipa::path(
    get,
    path = "/roles/{id}",
    tag = "roles",
    params(
        ("id" = String, Path, description = "ロールID")
    ),
    responses(
        (status = 200, description = "ロール情報の取得に成功", body = RoleResponse),
        (status = 404, description = "ロールが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_role(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let role = Role::find_by_id(id).one(&db).await.unwrap();

    if let Some(role) = role {
        Ok((StatusCode::OK, Json(RoleResponse::from(role))))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

#[derive(serde::Deserialize, ToSchema)]
pub struct CreateRole {
    pub custom_id: String,
    pub name: String,
    pub permission: i32,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
}

/// 新しいロールを作成するための関数
/// システム専用
#[utoipa::path(
    post,
    path = "/roles/{id}",
    tag = "roles",
    request_body = CreateRole,
    responses(
        (status = 201, description = "ロールの作成に成功", body = RoleResponse),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn create_role(
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
    Ok((StatusCode::CREATED, Json(RoleResponse::from(res))))
}

#[utoipa::path(
    put,
    path = "/roles/{id}",
    tag = "roles",
    params(
        ("id" = String, Path, description = "ロールID")
    ),
    request_body = CreateRole,
    responses(
        (status = 200, description = "ロールの更新に成功", body = RoleResponse),
        (status = 404, description = "ロールが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn put_role(
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
        return Ok((StatusCode::OK, Json(RoleResponse::from(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize, ToSchema)]
pub struct UpdateRole {
    pub name: Option<String>,
    pub custom_id: Option<String>,
    pub permission: Option<i32>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
}

/// ロールを差分アップデートするための関数
#[utoipa::path(
    patch,
    path = "/roles/{id}",
    tag = "roles",
    params(
        ("id" = String, Path, description = "ロールID")
    ),
    request_body = UpdateRole,
    responses(
        (status = 200, description = "ロールの部分更新に成功", body = RoleResponse),
        (status = 404, description = "ロールが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn patch_update_role(
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
        return Ok((StatusCode::OK, Json(RoleResponse::from(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

/// ロールを削除するための関数
#[utoipa::path(
    delete,
    path = "/roles/{id}",
    tag = "roles",
    params(
        ("id" = String, Path, description = "ロールID")
    ),
    responses(
        (status = 204, description = "ロールの削除に成功"),
        (status = 404, description = "ロールが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn delete_role(
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
