use axum::{
    extract::{Request, State},
    http::StatusCode,
    middleware::Next,
    response::Response,
};
use sea_orm::*;

use crate::{db::DbConn, models::session};

/// 認証されたユーザー情報を保持する構造体
#[derive(Clone, Debug)]
pub struct AuthUser {
    pub user_id: String,
    pub session_id: String,
    pub is_system: Option<bool>,
}

/// セッショントークンからユーザーを認証するミドルウェア
pub async fn auth_middleware(
    State(db): State<DbConn>,
    mut req: Request,
    next: Next,
) -> Result<Response, StatusCode> {
    // swagger-uiのみ許可
    if req.uri().path().starts_with("/swagger-ui")
        || req.uri().path().starts_with("/api-docs")
        || req.uri().path().starts_with("/openapi.json")
    {
        return Ok(next.run(req).await);
    }

    // ヘッダーからAPI_KEYを取得
    let api_key = req.headers().get("x-api-key").and_then(|h| h.to_str().ok());

    if (api_key.is_some() && api_key.unwrap() == std::env::var("API_KEY").unwrap_or_default()) {
        // API_KEYが一致する場合、システムユーザーとして認証情報を設定
        req.extensions_mut().insert(AuthUser {
            user_id: "system".to_string(),
            session_id: "system".to_string(),
            is_system: Some(true),
        });
        return Ok(next.run(req).await);
    }

    // Cookie ヘッダーから unique-sid を取得
    let cookie_header = req
        .headers()
        .get("Cookie")
        .and_then(|h| h.to_str().ok())
        .ok_or(StatusCode::UNAUTHORIZED)?;

    // unique-sid Cookie を抽出
    let token = cookie_header
        .split(';')
        .find_map(|cookie| {
            let cookie = cookie.trim();
            cookie.strip_prefix("unique-sid=")
        })
        .ok_or(StatusCode::UNAUTHORIZED)?;

    // セッション検証
    let found_session = session::Entity::find()
        .filter(session::Column::Id.eq(token))
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let session_model = found_session.ok_or(StatusCode::UNAUTHORIZED)?;

    // セッションが有効か確認
    if !session_model.is_enable {
        return Err(StatusCode::UNAUTHORIZED);
    }

    // リクエストに認証情報を追加
    req.extensions_mut().insert(AuthUser {
        user_id: session_model.user_id.clone(),
        session_id: session_model.id.clone(),
        is_system: Some(false),
    });

    Ok(next.run(req).await)
}
