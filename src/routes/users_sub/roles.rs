use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::role::Entity as Role,
    models::user::Entity as User,
    routes::{common_dtos::array_dto::ApiResponse, roles::RoleResponse},
};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{uid}/roles", get(get_all_roles))
        .route("/users/{uid}/roles/{id}", delete(delete_role).put(put_role))
    //.merge(users_sub::books::routes())
}

/// すべてのロールを取得するための関数
#[utoipa::path(
    get,
    path = "/users/{uid}/roles",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID")
    ),
    responses(
        (status = 200, description = "ユーザーのロール一覧取得成功"),
        (status = 404, description = "ユーザーが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_all_roles(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(
        &auth_user,
        Permission::PERMISSION_MANAGE,
        &uid,
        &db,
    )
    .await?;

    let user = User::find_by_id(uid).one(&db).await.unwrap();
    if let Some(user) = user {
        let roles = user.find_related(Role).all(&db).await.unwrap();
        let responses: Vec<RoleResponse> = roles.into_iter().map(RoleResponse::from).collect();
        return Ok((StatusCode::OK, Json(ApiResponse { data: responses })));
    }
    Err(StatusCode::NOT_FOUND)
}

// ロール付与
#[utoipa::path(
    put,
    path = "/users/{uid}/roles/{id}",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID"),
        ("id" = String, Path, description = "ロールID")
    ),
    responses(
        (status = 201, description = "ロール付与成功", body = crate::routes::roles::RoleResponse),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ユーザーまたはロールが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn put_role(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::PERMISSION_MANAGE, &db).await?;

    // user と role を同時に取りに行く（並列）
    let (user_res, role_res) = futures::join!(
        User::find_by_id(uid.clone()).one(&db),
        Role::find_by_id(id.clone()).one(&db)
    );

    match (user_res.unwrap(), role_res.unwrap()) {
        (Some(user), Some(role)) => {
            let user_role = crate::models::user_role::ActiveModel {
                user_id: Set(user.id.clone()),
                role_id: Set(role.id.clone()),
                ..Default::default()
            };
            // 既にある場合はエラーになるかもしれないから、必要なら重複チェックを追加
            let _ = user_role.insert(&db).await.unwrap();
            Ok((StatusCode::CREATED, Json(RoleResponse::from(role))))
        }
        _ => Err(StatusCode::NOT_FOUND),
    }
}

/// ユーザーを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
#[utoipa::path(
    delete,
    path = "/users/{uid}/roles/{id}",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID"),
        ("id" = String, Path, description = "ロールID")
    ),
    responses(
        (status = 204, description = "ロール削除成功"),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ユーザーまたはロールが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn delete_role(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::PERMISSION_MANAGE, &db).await?;

    // まず role の存在は確認しておくとレスポンスに role を返せる（現在の実装と同じ振る舞い）
    if let Some(role) = Role::find_by_id(id.clone()).one(&db).await.unwrap() {
        // 中間テーブルの該当行を直接削除
        let res = crate::models::user_role::Entity::delete_many()
            .filter(
                crate::models::user_role::Column::UserId
                    .eq(uid.clone())
                    .and(crate::models::user_role::Column::RoleId.eq(id.clone())),
            )
            .exec(&db)
            .await
            .unwrap();

        if res.rows_affected > 0 {
            return Ok((StatusCode::NO_CONTENT, Json(RoleResponse::from(role))));
        } else {
            return Err(StatusCode::NOT_FOUND);
        }
    }
    Err(StatusCode::NOT_FOUND)
}
