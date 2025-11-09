use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use sea_orm::*;
use serde_json;

use crate::models::discord::{self, Entity as Discord};
use crate::models::user::{self, Entity as User};
//use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users/{id}/discord", get(get_all_discord).put(put_discord))
        .route("/users/{id}/discord/{discord_id}", delete(delete_discord))
    //.merge(users_sub::books::routes())
}

/// ユーザーのDiscordアカウント一覧を取得するための関数
async fn get_all_discord(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let user = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = user {
        let discord_accounts = user
            .find_related(crate::models::discord::Entity)
            .all(&db)
            .await
            .unwrap();
        return (
            StatusCode::OK,
            Json(serde_json::json!({ "data": discord_accounts })),
        );
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}

#[derive(serde::Deserialize)]
struct CreateDiscord {
    pub discord_id: String,
    pub discord_customid: String,
}

/// ユーザーのDiscordアカウントを紐つけるための関数
async fn put_discord(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<CreateDiscord>,
) -> impl IntoResponse {
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let am = discord::ActiveModel {
            discord_id: Set(payload.discord_id),
            discord_customid: Set(payload.discord_customid),
            user_id: Set(user.id),
            ..Default::default()
        };
        let res = am.insert(&db).await.unwrap();
        return (StatusCode::CREATED, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<discord::Model>>(None))
}

/// ユーザーのDiscordアカウントの紐付けを解除するための関数
async fn delete_discord(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Path(discord_id): Path<String>,
) -> impl IntoResponse {
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
            return (StatusCode::NO_CONTENT, Json(serde_json::Value::Null));
        }
        return (StatusCode::NOT_FOUND, Json(serde_json::Value::Null));
    }
    (StatusCode::NOT_FOUND, Json(serde_json::Value::Null))
}
