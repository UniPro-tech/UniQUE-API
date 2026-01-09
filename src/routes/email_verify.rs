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
use serde_json::json;

use crate::{
    middleware::auth::AuthUser,
    models::email_verification,
};

pub fn routes() -> Router<DbConn> {
    Router::new().route(
        "/email_verify/{id}",
        get(get_email_verifications).delete(delete_email_verification),
    )
    //.merge(users_sub::books::routes())
}

async fn get_email_verifications(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    if id.chars().all(|c| c.is_numeric()) {
        let verifications = email_verification::Entity::find_by_id(id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?)
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
async fn delete_email_verification(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    if id.chars().all(|c| c.is_numeric()) {
        email_verification::Entity::delete_by_id(id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?)
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
        am.delete(&db).await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        return Ok((StatusCode::NO_CONTENT, Json(serde_json::Value::Null)));
    }

    Err(StatusCode::NOT_FOUND)
}
