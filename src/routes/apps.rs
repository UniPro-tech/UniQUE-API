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

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::{
        app::{self, Entity as App},
        user_app,
    },
};
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
async fn get_all_apps(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    let apps = App::find().all(&db).await.unwrap();

    // client_secretは常に除外
    let sanitized: Vec<serde_json::Value> = apps
        .into_iter()
        .map(|app| {
            let mut v = serde_json::to_value(&app).unwrap();
            if let Some(obj) = v.as_object_mut() {
                obj.remove("client_secret");
            }
            v
        })
        .collect();

    Ok(Json(serde_json::json!({ "data": sanitized })))
}

/// 特定のアプリケーションを取得するための関数
/// 誰でもアクセス可能だが、所有者以外はclient_secretを見られない
async fn get_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    let app = App::find_by_id(id.clone())
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(app) = app {
        // 所有者かどうかを確認
        let is_owner = user_app::Entity::find()
            .filter(user_app::Column::AppId.eq(&id))
            .filter(user_app::Column::UserId.eq(&auth_user.user_id))
            .one(&db)
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
            .is_some();

        let mut res = serde_json::to_value(&app).unwrap();

        // 所有者でない場合はclient_secretを除外
        if !is_owner {
            if let Some(obj) = res.as_object_mut() {
                obj.remove("client_secret");
            }
        }

        Ok((StatusCode::OK, Json(Some(res))))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

#[derive(serde::Deserialize)]
struct CreateApp {
    pub name: String,
    pub is_enable: Option<bool>,
}

/// 新しいアプリケーションを作成するための関数
/// システム専用
async fn create_app(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateApp>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::APP_CREATE, &db).await?;

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
    Ok((StatusCode::CREATED, Json(res)))
}

async fn put_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateApp>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::APP_UPDATE, &db).await?;

    let found = app::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: app::ActiveModel = user.into();
        am.name = Set(payload.name);
        am.is_enable = Set(payload.is_enable);
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
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
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<UpdateApp>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::APP_UPDATE, &db).await?;

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
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

/// アプリケーションを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::APP_DELETE, &db).await?;

    let found = App::find_by_id(id).one(&db).await.unwrap();
    if let Some(app) = found {
        let am: app::ActiveModel = app.into();
        am.delete(&db).await.unwrap();
        return Ok((StatusCode::NO_CONTENT, Json::<Option<app::Model>>(None)));
    }
    Err(StatusCode::NOT_FOUND)
}
