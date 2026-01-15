use crate::{
    constants::permissions::Permission, middleware::auth::AuthUser, models::user::Entity as User,
    routes::users_sub::permissions::PermissionsResponse,
};
use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users/{id}/permissions", get(get_permissions_bit))
}

/// ユーザーのロール一覧を取得し権限bitを合成する
#[utoipa::path(
    get,
    path = "/users/{id}/permissions",
    tag = "users",
    params(
        ("id" = String, Path, description = "ユーザーID")
    ),
    responses(
        (status = 200, description = "権限情報取得成功", body = PermissionsResponse),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ユーザーが見つからない")
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
    // Self-only access OR read permission
    if auth_user.user_id != id {
        let user_roles = crate::models::user_role::Entity::find()
            .filter(crate::models::user_role::Column::UserId.eq(&auth_user.user_id))
            .find_with_related(crate::models::role::Entity)
            .all(&db)
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

        let has_read_permission = user_roles.iter().any(|(_, roles)| {
            roles
                .iter()
                .any(|r| (r.permission as i64 & Permission::USER_READ.bits() as i64) != 0)
        });

        if !has_read_permission {
            return Err(StatusCode::FORBIDDEN);
        }
    }

    let user = User::find_by_id(id)
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(user) = user {
        let roles = user
            .find_related(crate::models::role::Entity)
            .all(&db)
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

        let mut permissions_bit: i64 = 0;
        for role in roles {
            permissions_bit |= role.permission as i64;
        }
        // bitだけではなく、テキストベースでも返す
        let mut permissions_text = Vec::new();

        let perm_u32 = permissions_bit as u32;
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
                permissions_bit,
                permissions_text,
            }),
        ));
    }
    Err(StatusCode::NOT_FOUND)
}
