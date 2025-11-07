use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;

use crate::models::role::{self, Entity as Role};
use crate::models::user::Entity as User;

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{uid}/roles", get(get_all_roles))
        .route("/users/{uid}/roles/{id}", delete(delete_role).put(put_role))
    //.merge(users_sub::books::routes())
}

/// すべてのロールを取得するための関数
async fn get_all_roles(State(db): State<DbConn>, Path(uid): Path<String>) -> impl IntoResponse {
    let user = User::find_by_id(uid).one(&db).await.unwrap();
    if let Some(user) = user {
        let roles = user.find_related(Role).all(&db).await.unwrap();
        return (StatusCode::OK, Json(serde_json::json!({ "data": roles })));
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}

// ロール付与
async fn put_role(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    Path(id): Path<String>,
) -> impl IntoResponse {
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
            (StatusCode::CREATED, Json(Some(role)))
        }
        _ => (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None)),
    }
}

/// ユーザーを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_role(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    Path(id): Path<String>,
) -> impl IntoResponse {
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
            return (StatusCode::NO_CONTENT, Json(Some(role)));
        } else {
            return (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None));
        }
    }
    (StatusCode::NOT_FOUND, Json::<Option<role::Model>>(None))
}
