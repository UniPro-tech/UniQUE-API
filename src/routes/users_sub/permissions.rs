use crate::models::role::Entity as Role;
use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
};
use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde::Serialize;
use utoipa::ToSchema;

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct PermissionsResponse {
    pub permissions_bit: i64,
    pub permissions_text: Vec<String>,
}

pub fn routes() -> Router<DbConn> {
    Router::new().route("/roles/{id}/permissions", get(get_permissions_bit))
}

/// ユーザーのロール一覧を取得し権限bitを合成する
#[utoipa::path(
    get,
    path = "/roles/{id}/permissions",
    tag = "roles",
    params(
        ("id" = String, Path, description = "ロールID")
    ),
    responses(
        (status = 200, description = "権限情報取得成功", body = PermissionsResponse),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ロールが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_permissions_bit(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // Check if the auth_user has permission to view roles
    let user_roles = crate::models::user_role::Entity::find()
        .filter(crate::models::user_role::Column::UserId.eq(&auth_user.user_id))
        .find_with_related(Role)
        .all(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let check_permission = permission_check::get_user_permissions(&auth_user, &db)
        .await?
        .contains(Permission::ROLE_MANAGE);
    if !check_permission
        || user_roles
            .iter()
            .all(|(_, roles)| roles.iter().all(|role| role.id != id))
    {
        return Err(StatusCode::FORBIDDEN);
    }

    let role = Role::find_by_id(id)
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(role) = role {
        // bitだけではなく、テキストベースでも返す
        let mut permissions_text = Vec::new();

        let perm_u32 = role.permission as u32;
        let (mut known_names, known_mask) = Permission::names_from_bits(perm_u32);
        permissions_text.append(&mut known_names);

        // 未定義のビットは PERMISSION_<index> でフォールバック
        let remaining = perm_u32 & !known_mask;
        for i in 0..32 {
            if (remaining & (1u32 << i)) != 0 {
                permissions_text.push(format!("PERMISSION_{}", i));
            }
        }
        return Ok((
            StatusCode::OK,
            Json(PermissionsResponse {
                permissions_bit: role.permission as i64,
                permissions_text,
            }),
        ));
    }
    Err(StatusCode::NOT_FOUND)
}
