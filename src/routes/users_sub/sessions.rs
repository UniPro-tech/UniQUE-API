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
    models::{
        session::{self, Entity as Session},
        user::Entity as User,
    },
    routes::{
        common_dtos::array_dto::ApiResponse, sessions::SessionResponse, users::PublicUserResponse,
    },
};

/// =======================
/// Router
/// =======================

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{uid}/sessions", get(get_all_sessions))
        .route(
            "/users/{uid}/sessions/{id}",
            get(get_session).delete(delete_session),
        )
}

/// =======================
/// handlers
/// =======================

/// ユーザーの全セッション取得
async fn get_all_sessions(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::SESSION_MANAGE, &uid, &db)
        .await?;

    let user = User::find_by_id(&uid).one(&db).await.unwrap();
    let Some(user) = user else {
        return Err(StatusCode::NOT_FOUND);
    };

    // セッション取得
    let sessions = user.find_related(Session).all(&db).await.unwrap();

    // DTO 変換
    let responses: Vec<SessionResponse> = sessions
        .into_iter()
        .map(|session| SessionResponse {
            id: session.id,
            created_at: session.created_at,
            expires_at: session.expires_at,
            ip_address: session.ip_address,
            user_agent: session.user_agent,
            is_enable: session.is_enable,
            user: PublicUserResponse {
                id: user.id.clone(),
                custom_id: user.custom_id.clone(),
                period: user.period.clone(),
                name: user.name.clone(),
                email: user.email.clone(),
                is_enable: user.is_enable,
            },
        })
        .collect();

    Ok((StatusCode::OK, Json(ApiResponse { data: responses })))
}

/// 特定セッション取得（単体なので ApiResponse で包まない）
async fn get_session(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::SESSION_MANAGE, &uid, &db)
        .await?;

    let joined = Session::find()
        .filter(session::Column::Id.eq(&id))
        .filter(session::Column::UserId.eq(&uid))
        .find_with_related(User)
        .all(&db)
        .await
        .unwrap();

    let Some((session, mut users)) = joined.into_iter().next() else {
        return Err(StatusCode::NOT_FOUND);
    };
    let user = users.pop().unwrap();

    let response = SessionResponse {
        id: session.id,
        created_at: session.created_at,
        expires_at: session.expires_at,
        ip_address: session.ip_address,
        user_agent: session.user_agent,
        is_enable: session.is_enable,
        user: PublicUserResponse {
            id: user.id,
            custom_id: user.custom_id,
            period: user.period,
            name: user.name,
            email: user.email,
            is_enable: user.is_enable,
        },
    };

    Ok((StatusCode::OK, Json(response)))
}

/// セッション削除
async fn delete_session(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::SESSION_MANAGE, &uid, &db)
        .await?;

    let found = Session::find()
        .filter(session::Column::Id.eq(id))
        .filter(session::Column::UserId.eq(uid))
        .one(&db)
        .await
        .unwrap();

    let Some(session) = found else {
        return Err(StatusCode::NOT_FOUND);
    };

    let am: session::ActiveModel = session.into();
    am.delete(&db).await.unwrap();

    Ok(StatusCode::NO_CONTENT)
}
