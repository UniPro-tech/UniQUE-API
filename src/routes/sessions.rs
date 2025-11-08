use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;

use crate::models::session::{self, Entity as Session};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/sessions", get(get_all_sessions))
        .route("/sessions/{id}", get(get_session).delete(delete_session))
}

/// すべてのセッションを取得するための関数
async fn get_all_sessions(State(db): State<DbConn>) -> Json<serde_json::Value> {
    let sessions = Session::find().all(&db).await.unwrap();
    Json(serde_json::json!({ "data": sessions }))
}

/// 特定のセッションを取得するための関数
async fn get_session(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    // セッションと関連のデータを結合して取得する（例: user を関連として取得する場合）
    let joined = Session::find()
        .filter(session::Column::Id.eq(id.clone()))
        .find_with_related(crate::models::user::Entity)
        .all(&db)
        .await
        .unwrap();
    // find_with_related は Vec<(session::Model, Vec<related::Model>)> を返す
    if let Some((session, related)) = joined.into_iter().next() {
        let mut body = serde_json::json!(session);
        body["user"] = serde_json::to_value(&related[0]).unwrap();
        // "roles" フィールドを追加
        let user_roles = crate::models::user_role::Entity::find()
            .filter(crate::models::user_role::Column::UserId.eq(&session.user_id))
            .find_with_related(crate::models::role::Entity)
            .all(&db)
            .await
            .unwrap();
        let roles: Vec<crate::models::role::Model> = user_roles
            .into_iter()
            .flat_map(|(_, roles)| roles)
            .collect();
        body["user"]["roles"] = serde_json::to_value(&roles).unwrap();
        // "user_id" フィールドを削除
        body.as_object_mut().unwrap().remove("user_id");
        return (StatusCode::OK, Json(body));
    }
    (StatusCode::NOT_FOUND, Json(serde_json::json!(null)))
}

/// セッションを削除するための関数
async fn delete_session(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let found = Session::find_by_id(id).one(&db).await.unwrap();
    if let Some(session) = found {
        let am: session::ActiveModel = session.into();
        am.delete(&db).await.unwrap();
        return (StatusCode::NO_CONTENT, Json::<Option<session::Model>>(None));
    }
    (StatusCode::NOT_FOUND, Json::<Option<session::Model>>(None))
}
