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
use ulid::Ulid;
use utoipa::ToSchema;

use crate::{
    middleware::auth::AuthUser,
    models::{
        email_verification,
        user::{self, Entity as User},
    },
    routes::email_verify::EmailVerificationResponse,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct EmailVerificationResponse {
    pub id: Option<i32>,
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
            id: Some(model.id),
            user_id: model.user_id,
            verification_code: model.verification_code,
            created_at: model.created_at,
            expires_at: model.expires_at,
        }
    }
}

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{uid}/email_verify", post(post_challenge))
        .route(
            "/users/{uid}/email_verify/{id}",
            get(get_email_verifications).delete(delete_email_verification),
        )
    //.merge(users_sub::books::routes())
}

#[utoipa::path(
    get,
    path = "/users/{uid}/email_verify/{id}",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID"),
        ("id" = String, Path, description = "Email検証ID or 検証コード")
    ),
    responses(
        (status = 200, description = "Email検証情報取得成功", body = EmailVerificationResponse),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "Email検証が見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn get_email_verifications(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // Self-only access
    if auth_user.user_id != uid {
        return Err(StatusCode::FORBIDDEN);
    }

    let found = User::find_by_id(uid)
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(user) = found {
        if id.chars().all(|c| c.is_numeric()) {
            let verifications = email_verification::Entity::find_by_id(
                id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?,
            )
            .filter(email_verification::Column::UserId.eq(user.id))
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
            .filter(email_verification::Column::UserId.eq(user.id))
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
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize, ToSchema)]
pub struct CreateVerifyChallenge {
    #[schema(value_type = String, format = "date-time")]
    pub expires_at: Option<chrono::DateTime<chrono::Utc>>,
}

/// ユーザーのEmail検証チャレンジを作成するための関数
#[utoipa::path(
    post,
    path = "/users/{uid}/email_verify",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID")
    ),
    request_body = CreateVerifyChallenge,
    responses(
        (status = 201, description = "Email検証チャレンジ作成成功", body = EmailVerificationResponse),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "ユーザーが見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn post_challenge(
    State(db): State<DbConn>,
    Path(uid): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateVerifyChallenge>,
) -> Result<impl IntoResponse, StatusCode> {
    // Self-only access
    if auth_user.user_id != uid {
        return Err(StatusCode::FORBIDDEN);
    }

    let found = user::Entity::find_by_id(uid)
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

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
            Ok(res) => {
                return Ok((
                    StatusCode::CREATED,
                    Json(EmailVerificationResponse::from(res)),
                ));
            }
            Err(_) => {
                return Err(StatusCode::INTERNAL_SERVER_ERROR);
            }
        }
    }
    Err(StatusCode::NOT_FOUND)
}

/// ユーザーのEmail検証チャレンジを削除するための関数
#[utoipa::path(
    delete,
    path = "/users/{uid}/email_verify/{id}",
    tag = "users",
    params(
        ("uid" = String, Path, description = "ユーザーID"),
        ("id" = String, Path, description = "Email検証ID or 検証コード")
    ),
    responses(
        (status = 204, description = "Email検証削除成功"),
        (status = 403, description = "アクセス権限なし"),
        (status = 404, description = "Email検証が見つからない")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn delete_email_verification(
    State(db): State<DbConn>,
    Path((uid, id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // Self-only access
    if auth_user.user_id != uid {
        return Err(StatusCode::FORBIDDEN);
    }

    let found = User::find_by_id(uid)
        .one(&db)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    if let Some(user) = found {
        if id.chars().all(|c| c.is_numeric()) {
            email_verification::Entity::delete_by_id(
                id.parse::<i32>().map_err(|_| StatusCode::BAD_REQUEST)?,
            )
            .filter(email_verification::Column::UserId.eq(user.id))
            .exec(&db)
            .await
            .map_err(|e| match e {
                DbErr::RecordNotFound(_) => StatusCode::NOT_FOUND,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            })?;

            return Ok((
                StatusCode::NO_CONTENT,
                Json::<Option<email_verification::Model>>(None),
            ));
        }
        let found = email_verification::Entity::find()
            .filter(email_verification::Column::UserId.eq(user.id))
            .filter(email_verification::Column::VerificationCode.eq(id))
            .one(&db)
            .await
            .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
        if let Some(found) = found {
            let am: email_verification::ActiveModel = found.into();
            am.delete(&db)
                .await
                .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
            return Ok((
                StatusCode::NO_CONTENT,
                Json::<Option<email_verification::Model>>(None),
            ));
        }
    }

    Err(StatusCode::NOT_FOUND)
}
