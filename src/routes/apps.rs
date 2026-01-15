use axum::{
    Json, Router,
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use chrono::Utc;
use sea_orm::*;
use serde::Serialize;
use serde_json;
use sha2::Digest;
use ulid::Ulid;
use utoipa::{IntoParams, ToSchema};

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::{
        app::{self, Entity as App},
        user_app,
    },
    routes::common_dtos::array_dto::ApiResponse,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct AppResponse {
    pub id: String,
    pub name: String,
    pub is_enable: Option<bool>,
    pub created_at: Option<chrono::DateTime<Utc>>,
    pub updated_at: Option<chrono::DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub client_secret: Option<String>,
}
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
/// ?all=trueがある場合、APP_READ権限が必要です
#[derive(serde::Deserialize, ToSchema, IntoParams)]
pub struct GetAllAppsQuery {
    #[serde(default)]
    pub all: bool,
}

#[utoipa::path(
    get,
    path = "/apps",
    tag = "apps",
    params(
        GetAllAppsQuery
    ),
    responses(
        (status = 200, description = "アプリケーション一覧の取得に成功", body = ApiResponse<Vec<AppResponse>>),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_all_apps(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
    Query(query): Query<GetAllAppsQuery>,
) -> Result<impl IntoResponse, StatusCode> {
    // all=trueの場合はAPP_READ権限をチェック
    if query.all {
        permission_check::require_permission(&auth_user, Permission::APP_READ, &db).await?;
    }

    let apps = App::find().all(&db).await.unwrap();

    // client_secretは常に除外
    let responses: Vec<AppResponse> = apps
        .into_iter()
        .map(|app| AppResponse {
            id: app.id,
            name: app.name,
            is_enable: app.is_enable,
            created_at: app.created_at,
            updated_at: app.updated_at,
            client_secret: None, // 常に除外
        })
        .collect();

    Ok((StatusCode::OK, Json(ApiResponse { data: responses })))
}

/// 特定のアプリケーションを取得するための関数
/// 誰でもアクセス可能だが、所有者以外はclient_secretを見られない
#[utoipa::path(
    get,
    path = "/apps/{id}",
    tag = "apps",
    params(
        ("id" = String, Path, description = "アプリID")
    ),
    responses(
        (status = 200, description = "アプリ情報の取得に成功", body = AppResponse),
        (status = 404, description = "アプリが見つからない"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_app(
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

        let response = AppResponse {
            id: app.id,
            name: app.name,
            is_enable: app.is_enable,
            created_at: app.created_at,
            updated_at: app.updated_at,
            client_secret: if is_owner {
                Some(app.client_secret)
            } else {
                None
            },
        };

        Ok((StatusCode::OK, Json(response)))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

#[derive(serde::Deserialize, ToSchema)]
pub struct CreateApp {
    pub name: String,
    pub is_enable: Option<bool>,
}

/// 新しいアプリケーションを作成するための関数
#[utoipa::path(
    post,
    path = "/apps/{id}",
    tag = "apps",
    request_body = CreateApp,
    responses(
        (status = 201, description = "アプリケーションの作成に成功", body = AppResponse),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn create_app(
    State(db): State<DbConn>,
    _auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateApp>,
) -> Result<impl IntoResponse, StatusCode> {
    let client_secret = {
        let uuid = uuid::Uuid::new_v4().to_string();
        let hash = sha2::Sha256::digest(uuid.as_bytes());
        hex::encode(hash)
    };

    let am = app::ActiveModel {
        id: Set(Ulid::new().to_string()),
        name: Set(payload.name),
        created_at: Set(Some(Utc::now())),
        updated_at: Set(Some(Utc::now())),
        is_enable: Set(Some(payload.is_enable.unwrap_or(true))),
        client_secret: Set(client_secret.clone()),
    };
    let res = am.insert(&db).await.unwrap();

    let response = AppResponse {
        id: res.id,
        name: res.name,
        is_enable: res.is_enable,
        created_at: res.created_at,
        updated_at: res.updated_at,
        client_secret: Some(client_secret), // 作成時は返す
    };

    Ok((StatusCode::CREATED, Json(response)))
}

#[utoipa::path(
    put,
    path = "/apps/{id}",
    tag = "apps",
    params(
        ("id" = String, Path, description = "アプリID")
    ),
    request_body = CreateApp,
    responses(
        (status = 200, description = "アプリの更新に成功", body = AppResponse),
        (status = 404, description = "アプリが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn put_app(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateApp>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::APP_UPDATE, &db).await?;

    let found = app::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(app_model) = found {
        let mut am: app::ActiveModel = app_model.into();
        am.name = Set(payload.name);
        am.is_enable = Set(payload.is_enable);
        am.updated_at = Set(Some(Utc::now()));
        let res = am.update(&db).await.unwrap();

        let response = AppResponse {
            id: res.id,
            name: res.name,
            is_enable: res.is_enable,
            created_at: res.created_at,
            updated_at: res.updated_at,
            client_secret: None, // 更新時は返さない
        };

        return Ok((StatusCode::OK, Json(response)));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize, ToSchema)]
pub struct UpdateApp {
    pub name: Option<String>,
    pub is_enable: Option<bool>,
}

/// アプリケーションを差分アップデートするための関数
#[utoipa::path(
    patch,
    path = "/apps/{id}",
    tag = "apps",
    params(
        ("id" = String, Path, description = "アプリID")
    ),
    request_body = UpdateApp,
    responses(
        (status = 200, description = "アプリの部分更新に成功", body = AppResponse),
        (status = 404, description = "アプリが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn patch_update_app(
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

        let response = AppResponse {
            id: res.id,
            name: res.name,
            is_enable: res.is_enable,
            created_at: res.created_at,
            updated_at: res.updated_at,
            client_secret: None,
        };

        return Ok((StatusCode::OK, Json(response)));
    }
    Err(StatusCode::NOT_FOUND)
}

/// アプリケーションを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
#[utoipa::path(
    delete,
    path = "/apps/{id}",
    tag = "apps",
    params(
        ("id" = String, Path, description = "アプリID")
    ),
    responses(
        (status = 204, description = "アプリの削除に成功"),
        (status = 404, description = "アプリが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn delete_app(
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
