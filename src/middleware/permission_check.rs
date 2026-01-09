use axum::{
    extract::{Request, State, rejection::JsonRejection},
    http::StatusCode,
    middleware::Next,
    response::{IntoResponse, Response},
};
use sea_orm::*;

use crate::{
    constants::permissions::Permission,
    db::DbConn,
    middleware::auth::AuthUser,
    models::{role, user::Entity as User},
};

/// 現在のユーザーの権限を取得
pub async fn get_user_permissions(
    auth_user: &AuthUser,
    db: &DbConn,
) -> Result<Permission, StatusCode> {
    let user = User::find_by_id(&auth_user.user_id)
        .one(db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
        .ok_or(StatusCode::UNAUTHORIZED)?;

    let roles = user
        .find_related(role::Entity)
        .all(db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let mut user_permissions = Permission::empty();
    for role_model in roles {
        if let Some(perm) = Permission::from_bits(role_model.permission as u32) {
            user_permissions |= perm;
        }
    }

    Ok(user_permissions)
}

/// 指定された権限を持っているかチェック
pub async fn require_permission(
    auth_user: &AuthUser,
    required: Permission,
    db: &DbConn,
) -> Result<(), StatusCode> {
    let user_permissions = get_user_permissions(auth_user, db).await?;
    if !user_permissions.contains(required) {
        return Err(StatusCode::FORBIDDEN);
    }
    Ok(())
}

/// 自分自身のリソースか、指定された権限を持っているかチェック
pub async fn require_permission_or_self(
    auth_user: &AuthUser,
    required: Permission,
    target_user_id: &str,
    db: &DbConn,
) -> Result<(), StatusCode> {
    // 自分自身のリソースならOK
    if auth_user.user_id == target_user_id {
        return Ok(());
    }

    // それ以外は権限チェック
    require_permission(auth_user, required, db).await
}
