use crate::constants::permissions::Permission;
use crate::models::user::Entity as User;
use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users/{id}/permissions", get(get_permissions_bit))
}

/// ユーザーのロール一覧を取得し権限bitを合成する
async fn get_permissions_bit(
    State(db): State<DbConn>,
    Path(id): Path<String>,
) -> impl IntoResponse {
    let user = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = user {
        let roles = user
            .find_related(crate::models::role::Entity)
            .all(&db)
            .await
            .unwrap();
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
        return (
            StatusCode::OK,
            Json(
                serde_json::json!({ "permissions_bit": permissions_bit, "permissions_text": permissions_text }),
            ),
        );
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
