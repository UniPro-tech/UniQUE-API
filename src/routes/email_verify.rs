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

impl From<email_verification::Model> for EmailVerificationResponse {
    fn from(model: email_verification::Model) -> Self {
        Self {
            id: model.id,
            user_id: model.user_id,
            verification_code: model.verification_code,
            created_at: model.created_at,
            expires_at: model.expires_at,
        }
    }
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
    _auth_user: axum::Extension<AuthUser>,
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
            return Ok((
                StatusCode::OK,
                Json(EmailVerificationResponse::from(verification)),
            ));
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
        return Ok((
            StatusCode::OK,
            Json(EmailVerificationResponse::from(verification)),
        ));
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
    _auth_user: axum::Extension<AuthUser>,
) -> Result<StatusCode, StatusCode> {
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

        return Ok(StatusCode::NO_CONTENT);
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
        return Ok(StatusCode::NO_CONTENT);
    }

    Err(StatusCode::NOT_FOUND)
}
