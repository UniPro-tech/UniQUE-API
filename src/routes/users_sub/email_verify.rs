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
use ulid::Ulid;

use crate::models::email_verification;
use crate::models::user::{self, Entity as User};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{uid}/email_verify", post(post_challenge))
        .route(
            "/users/{uid}/email_verify/{id}",
            get(get_email_verifications).delete(delete_email_verification),
        )
    //.merge(users_sub::books::routes())
}

async fn get_email_verifications(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
) -> impl IntoResponse {
    let found = match User::find_by_id(uid).one(&db).await {
        Ok(f) => f,
        Err(e) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({"error": e.to_string()})),
            );
        }
    };

    if let Some(user) = found {
        if id.chars().all(|c| c.is_numeric()) {
            let verifications =
                match email_verification::Entity::find_by_id(id.parse::<i32>().unwrap())
                    .filter(email_verification::Column::UserId.eq(user.id))
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
            .filter(email_verification::Column::UserId.eq(user.id))
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
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}

#[derive(serde::Deserialize)]
struct CreateVerifyChallenge {
    pub expires_at: Option<chrono::DateTime<chrono::Utc>>,
}

/// ユーザーのEmail検証チャレンジを作成するための関数
async fn post_challenge(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    Json(payload): Json<CreateVerifyChallenge>,
) -> impl IntoResponse {
    let found = match user::Entity::find_by_id(uid).one(&db).await {
        Ok(f) => f,
        Err(e) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({"error": e.to_string()})),
            );
        }
    };

    let expires_at = payload
        .expires_at
        .unwrap_or_else(|| Utc::now() + chrono::Duration::hours(24))
        .naive_utc();

    if let Some(user) = found {
        let challenge = email_verification::ActiveModel {
            user_id: Set(user.id.clone()),
            verification_code: Set(Ulid::new().to_string()),
            created_at: Set(Some(Utc::now().naive_utc())),
            expires_at: Set(expires_at),
            ..Default::default()
        };
        match challenge.insert(&db).await {
            Ok(res) => return (StatusCode::CREATED, Json(json!(res))),
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

/// ユーザーのEmail検証チャレンジを削除するための関数
async fn delete_email_verification(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
) -> impl IntoResponse {
    let found = match User::find_by_id(uid).one(&db).await {
        Ok(f) => f,
        Err(e) => {
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({"error": e.to_string()})),
            );
        }
    };

    if let Some(user) = found {
        if id.chars().all(|c| c.is_numeric()) {
            match email_verification::Entity::delete_by_id(id.parse::<i32>().unwrap())
                .filter(email_verification::Column::UserId.eq(user.id))
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
            .filter(email_verification::Column::UserId.eq(user.id))
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
    }

    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
