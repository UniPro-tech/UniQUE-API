use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use chrono::Utc;
use sea_orm::*;
use serde::Serialize;
use serde_json;
use serde_json::json;
use utoipa::ToSchema;

use crate::{middleware::auth::AuthUser, models::email_verification};

#[derive(Serialize, ToSchema)]
pub struct EmailVerificationResponse {
    pub id: i32,
    pub user_id: String,
    pub verification_code: String,
    #[schema(value_type = String, format = "date-time")]
    pub created_at: Option<chrono::NaiveDateTime>,
    #[schema(value_type = String, format = "date-time")]
    pub expires_at: chrono::NaiveDateTime,
}

pub fn routes() -> Router<DbConn> {
    Router::new().route(
        "/email_verify/{id}",
        get(get_email_verifications).delete(delete_email_verification),
    )
    //.merge(users_sub::books::routes())
}

#[utoipa::path(
    get,
    path = "/email_verify/{id}",
    tag = "email_verify",
    params(
        ("id" = String, Path, description = "Email検証ID or 検証コード")
    ),
    responses(
        (status = 200, description = "Email検証情報取得成功", body = serde_json::Value),
        (status = 404, description = "Email検証が見つからない"),
        (status = 500, description = "サーバーエラー")
    )
)]
pub async fn get_email_verifications(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    if id.chars().all(|c| c.is_numeric()) {
        let verifications = email_verification::Entity::find_by_id(
            id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?,
        )
        .filter(email_verification::Column::ExpiresAt.gt(Utc::now().naive_utc()))
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

        if let Some(verification) = verifications {
            return Ok((StatusCode::OK, Json(json!(verification))));
        }

        return Err(StatusCode::NOT_FOUND);
    }
    let verification = email_verification::Entity::find()
        .filter(email_verification::Column::VerificationCode.eq(id))
        .filter(email_verification::Column::ExpiresAt.gt(Utc::now().naive_utc()))
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(verification) = verification {
        return Ok((StatusCode::OK, Json(json!(verification))));
    }

    Err(StatusCode::NOT_FOUND)
}

/// ユーザーのEmail検証チャレンジを削除するための関数
#[utoipa::path(
    delete,
    path = "/email_verify/{id}",
    tag = "email_verify",
    params(
        ("id" = String, Path, description = "Email検証ID or 検証コード")
    ),
    responses(
        (status = 204, description = "Email検証削除成功"),
        (status = 404, description = "Email検証が見つからない"),
        (status = 500, description = "サーバーエラー")
    )
)]
pub async fn delete_email_verification(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    if id.chars().all(|c| c.is_numeric()) {
        email_verification::Entity::delete_by_id(
            id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?,
        )
        .exec(&db)
        .await
        .map_err(|e| match e {
            DbErr::RecordNotFound(_) => StatusCode::NOT_FOUND,
            _ => StatusCode::INTERNAL_SERVER_ERROR,
        })?;

        return Ok((StatusCode::NO_CONTENT, Json(serde_json::Value::Null)));
    }
    let found = email_verification::Entity::find()
        .filter(email_verification::Column::VerificationCode.eq(id))
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    if let Some(found) = found {
        let am: email_verification::ActiveModel = found.into();
        am.delete(&db)
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        return Ok((StatusCode::NO_CONTENT, Json(serde_json::Value::Null)));
    }

    Err(StatusCode::NOT_FOUND)
}
