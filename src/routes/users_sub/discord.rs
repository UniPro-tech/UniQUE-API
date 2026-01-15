use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde::Serialize;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::discord::{self, Entity as Discord},
    models::user::{self, Entity as User},
    routes::common_dtos::array_dto::ApiResponse,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize)]
pub struct DiscordResponse {
    pub discord_id: String,
    pub custom_id: String,
    pub user_id: String,
}

impl From<discord::Model> for DiscordResponse {
    fn from(discord: discord::Model) -> Self {
        Self {
            discord_id: discord.discord_id,
            custom_id: discord.custom_id,
            user_id: discord.user_id,
        }
    }
}

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{id}/discord", get(get_all_discord).put(put_discord))
        .route("/users/{id}/discord/{discord_id}", delete(delete_discord))
    //.merge(users_sub::books::routes())
}

/// ユーザーのDiscordアカウント一覧を取得するための関数
async fn get_all_discord(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::USER_READ, &id, &db)
        .await?;
    let user = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = user {
        let discord_accounts = user
            .find_related(crate::models::discord::Entity)
            .all(&db)
            .await
            .unwrap();
        let responses: Vec<DiscordResponse> = discord_accounts
            .into_iter()
            .map(DiscordResponse::from)
            .collect();
        return Ok((StatusCode::OK, Json(ApiResponse { data: responses })));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize)]
struct CreateDiscord {
    pub discord_id: String,
    pub custom_id: String,
}

/// ユーザーのDiscordアカウントを紐つけるための関数
async fn put_discord(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateDiscord>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::USER_UPDATE, &id, &db)
        .await?;
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let am = discord::ActiveModel {
            discord_id: Set(payload.discord_id),
            custom_id: Set(payload.custom_id),
            user_id: Set(user.id),
            ..Default::default()
        };
        let res = am.insert(&db).await.unwrap();
        return Ok((StatusCode::CREATED, Json(DiscordResponse::from(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

/// ユーザーのDiscordアカウントの紐付けを解除するための関数
async fn delete_discord(
    State(db): State<DbConn>,
    Path((id, discord_id)): Path<(String, String)>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::USER_UPDATE, &id, &db)
        .await?;
    let found = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let discord_account = Discord::find()
            .filter(discord::Column::UserId.eq(user.id))
            .filter(discord::Column::DiscordId.eq(discord_id))
            .one(&db)
            .await
            .unwrap();
        if let Some(discord_account) = discord_account {
            let am: discord::ActiveModel = discord_account.into();
            am.delete(&db).await.unwrap();
            return Ok((StatusCode::NO_CONTENT, Json::<Option<discord::Model>>(None)));
        }
        return Err(StatusCode::NOT_FOUND);
    }
    Err(StatusCode::NOT_FOUND)
}
