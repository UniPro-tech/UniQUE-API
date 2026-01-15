use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::{prelude::DateTimeUtc, *};
use serde::Serialize;
use utoipa::ToSchema;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::session::{self, Entity as Session},
    routes::{common_dtos::array_dto::ApiResponse, users::PublicUserResponse},
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct SessionResponse {
    pub id: String,
    #[schema(value_type = String, format = "date-time")]
    pub created_at: Option<DateTimeUtc>,
    #[schema(value_type = String, format = "date-time")]
    pub expires_at: Option<DateTimeUtc>,
    pub ip_address: String,
    pub user_agent: String,
    pub is_enable: bool,
    pub user: PublicUserResponse,
}

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/sessions", get(get_all_sessions))
        .route("/sessions/{id}", get(get_session).delete(delete_session))
}

/// すべてのセッションを取得するための関数
#[utoipa::path(
    get,
    path = "/sessions",
    tag = "sessions",
    responses(
        (status = 200, description = "セッション一覧の取得に成功", body = ApiResponse<Vec<SessionResponse>>),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_all_sessions(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::SESSION_MANAGE, &db).await?;

    // relatedでuserも取得する
    let joined = Session::find()
        .find_with_related(crate::models::user::Entity)
        .all(&db)
        .await
        .unwrap();

    let session_responses: Vec<SessionResponse> = joined
        .into_iter()
        .filter_map(|(session, users)| {
            users.first().map(|user| SessionResponse {
                id: session.id,
                created_at: session.created_at,
                expires_at: session.expires_at,
                ip_address: session.ip_address,
                user_agent: session.user_agent,
                is_enable: session.is_enable,
                user: PublicUserResponse::from(user.clone()),
            })
        })
        .collect();

    Ok((
        StatusCode::OK,
        Json(ApiResponse {
            data: session_responses,
        }),
    ))
}

/// 特定のセッションを取得するための関数
#[utoipa::path(
    get,
    path = "/sessions/{id}",
    tag = "sessions",
    params(
        ("id" = String, Path, description = "セッションID")
    ),
    responses(
        (status = 200, description = "セッション情報の取得に成功", body = SessionResponse),
        (status = 404, description = "セッションが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_session(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // セッションと関連のデータを結合して取得する（例: user を関連として取得する場合）
    let joined = Session::find()
        .filter(session::Column::Id.eq(id.clone()))
        .find_with_related(crate::models::user::Entity)
        .all(&db)
        .await
        .unwrap();
    // find_with_related は Vec<(session::Model, Vec<related::Model>)> を返す
    if let Some((session, related)) = joined.into_iter().next() {
        // 自分のセッションでない場合は SESSION_MANAGE 権限が必要
        if session.user_id != auth_user.user_id {
            permission_check::require_permission(&auth_user, Permission::SESSION_MANAGE, &db)
                .await?;
        }

        if let Some(user) = related.first() {
            let response = SessionResponse {
                id: session.id,
                created_at: session.created_at,
                expires_at: session.expires_at,
                ip_address: session.ip_address,
                user_agent: session.user_agent,
                is_enable: session.is_enable,
                user: PublicUserResponse::from(user.clone()),
            };
            return Ok((StatusCode::OK, Json(response)));
        }
    }
    Err(StatusCode::NOT_FOUND)
}

/// セッションを削除するための関数
#[utoipa::path(
    delete,
    path = "/sessions/{id}",
    tag = "sessions",
    params(
        ("id" = String, Path, description = "セッションID")
    ),
    responses(
        (status = 204, description = "セッションの削除に成功"),
        (status = 404, description = "セッションが見つからない"),
        (status = 403, description = "権限なし"),
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn delete_session(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    let found = Session::find_by_id(id).one(&db).await.unwrap();
    if let Some(session) = found {
        // 自分のセッションでない場合は SESSION_MANAGE 権限が必要
        if session.user_id != auth_user.user_id {
            permission_check::require_permission(&auth_user, Permission::SESSION_MANAGE, &db)
                .await?;
        }

        let am: session::ActiveModel = session.into();
        am.delete(&db).await.unwrap();
        return Ok((StatusCode::NO_CONTENT, Json::<Option<session::Model>>(None)));
    }
    Err(StatusCode::NOT_FOUND)
}
