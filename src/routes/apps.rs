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
use sha2::Digest;
use ulid::Ulid;

use crate::models::app::{self, Entity as App};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/apps", get(get_all_apps)).route(
        "/apps/{id}",
        get(get_app)
            .patch(patch_update_app)
            .delete(delete_app)
            .post(create_app)
            .put(put_app),
    )
    //.merge(users_sub::books::routes())
}

/// すべてのアプリケーションを取得するための関数
async fn get_all_apps(State(db): State<DbConn>) -> Json<serde_json::Value> {
    let apps = App::find().all(&db).await.unwrap();
    Json(serde_json::json!({ "data": apps }))
}

/// 特定のアプリケーションを取得するための関数
async fn get_app(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let app = App::find_by_id(id).one(&db).await.unwrap();

    if let Some(app) = app {
        (StatusCode::OK, Json(Some(app)))
    } else {
        (StatusCode::NOT_FOUND, Json::<Option<app::Model>>(None))
    }
}

#[derive(serde::Deserialize)]
struct CreateApp {
    pub name: String,
    pub is_enable: Option<bool>,
}

/// 新しいアプリケーションを作成するための関数
/// システム専用
async fn create_app(State(db): State<DbConn>, Json(payload): Json<CreateApp>) -> impl IntoResponse {
    let am = app::ActiveModel {
        id: Set(Ulid::new().to_string()),
        name: Set(payload.name),
        created_at: Set(Some(Utc::now())),
        updated_at: Set(Some(Utc::now())),
        is_enable: Set(Some(payload.is_enable.unwrap_or(true))),
        client_secret: Set({
            let uuid = uuid::Uuid::new_v4().to_string();
            let hash = sha2::Sha256::digest(uuid.as_bytes());
            hex::encode(hash)
        }),
    };
    let res = am.insert(&db).await.unwrap();
    (StatusCode::CREATED, Json(res))
}

async fn put_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<CreateApp>,
) -> impl IntoResponse {
    let found = app::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: app::ActiveModel = user.into();
        am.name = Set(payload.name);
        am.is_enable = Set(payload.is_enable);
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<app::Model>>(None))
}

#[derive(serde::Deserialize)]
struct UpdateApp {
    pub name: Option<String>,
    pub is_enable: Option<bool>,
}

/// アプリケーションを差分アップデートするための関数
async fn patch_update_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<UpdateApp>,
) -> impl IntoResponse {
    let found = app::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(app) = found {
        let mut am: app::ActiveModel = app.into();
        if let Some(name) = payload.name {
            am.name = Set(name);
        }
        if let Some(is_enable) = payload.is_enable {
            am.is_enable = Set(Some(is_enable));
        }
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<app::Model>>(None))
}

/// アプリケーションを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_app(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let found = App::find_by_id(id).one(&db).await.unwrap();
    if let Some(app) = found {
        let am: app::ActiveModel = app.into();
        am.delete(&db).await.unwrap();
        return (StatusCode::NO_CONTENT, Json::<Option<app::Model>>(None));
    }
    (StatusCode::NOT_FOUND, Json::<Option<app::Model>>(None))
}
