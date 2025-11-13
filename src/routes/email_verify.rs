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

use crate::models::email_verification;

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
) -> impl IntoResponse {
    if id.chars().all(|c| c.is_numeric()) {
        let verifications = match email_verification::Entity::find_by_id(id.parse::<i32>().unwrap())
            .filter(email_verification::Column::ExpiresAt.gt(Utc::now().naive_utc()))
            .one(&db)
            .await
        {
            Ok(v) => v,
            Err(e) => {
                return (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    Json(json!({"error": e.to_string()})),
                );
            }
        };

        if let Some(verification) = verifications {
            return (StatusCode::OK, Json(json!(verification)));
        }

        return (StatusCode::NOT_FOUND, Json(serde_json::Value::Null));
    }
    let verification = match email_verification::Entity::find()
        .filter(email_verification::Column::VerificationCode.eq(id))
        .filter(email_verification::Column::ExpiresAt.gt(Utc::now().naive_utc()))
        .one(&db)
        .await
    {
        Ok(v) => v,
        Err(e) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({"error": e.to_string()})),
            );
        }
    };

    if let Some(verification) = verification {
        return (StatusCode::OK, Json(json!(verification)));
    }

    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}

/// ユーザーのEmail検証チャレンジを削除するための関数
async fn delete_email_verification(
    State(db): State<DbConn>,
    Path(id): Path<String>,
) -> impl IntoResponse {
    if id.chars().all(|c| c.is_numeric()) {
        match email_verification::Entity::delete_by_id(id.parse::<i32>().unwrap())
            .exec(&db)
            .await
        {
            Ok(v) => v,
            Err(e) => match e {
                DbErr::RecordNotFound(_) => {
                    return (StatusCode::NOT_FOUND, Json(serde_json::Value::Null));
                }
                other => {
                    return (
                        StatusCode::INTERNAL_SERVER_ERROR,
                        Json(json!({"error": other.to_string()})),
                    );
                }
            },
        };

        return (StatusCode::NO_CONTENT, Json(serde_json::Value::Null));
    }
    let found = email_verification::Entity::find()
        .filter(email_verification::Column::VerificationCode.eq(id))
        .one(&db)
        .await
        .unwrap();
    if let Some(found) = found {
        let am: email_verification::ActiveModel = found.into();
        match am.delete(&db).await {
            Ok(_) => return (StatusCode::NO_CONTENT, Json(serde_json::Value::Null)),
            Err(e) => {
                return (
                    StatusCode::INTERNAL_SERVER_ERROR,
                    Json(json!({"error": e.to_string()})),
                );
            }
        }
    }

    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
