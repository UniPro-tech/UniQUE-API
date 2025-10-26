use axum::{
    Json, Router,
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::*,
};
use chrono::Utc;
use sea_orm::*;
use uuid::Uuid;

use crate::models::user::{self, Entity as User};
use crate::{db::DbConn, routes::users_sub};

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users", get(get_all_users))
        .route(
            "/users/{id}",
            get(get_user).patch(patch_update_user).delete(delete_user),
        )
        .merge(users_sub::books::routes())
}

async fn get_all_users(State(db): State<DbConn>) -> Json<serde_json::Value> {
    let users = User::find().all(&db).await.unwrap();
    Json(serde_json::json!({ "data": users }))
}

async fn get_user(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let user = User::find_by_id(id).one(&db).await.unwrap();

    if let Some(user) = user {
        (StatusCode::OK, Json(Some(user)))
    } else {
        (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
    }
}

#[derive(serde::Deserialize)]
struct UpdateUser {
    pub name: Option<String>,
    pub email: Option<String>,
    pub image: Option<String>,
    pub role: Option<String>,
    pub campus_id: Option<Uuid>,
}

async fn patch_update_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    Json(payload): Json<UpdateUser>,
) -> impl IntoResponse {
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        if let Some(name) = payload.name {
            am.name = Set(Some(name));
        }
        if let Some(email) = payload.email {
            am.email = Set(email);
        }
        if let Some(image) = payload.image {
            am.image = Set(Some(image));
        }
        if let Some(role) = payload.role {
            am.role = Set(Some(role));
        }
        if let Some(campus_id) = payload.campus_id {
            am.campus_id = Set(Some(campus_id));
        }
        am.updated_at = Set(Utc::now().naive_utc());
        let res = am.update(&db).await.unwrap();
        return (StatusCode::OK, Json(Some(res)));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}

async fn delete_user(State(db): State<DbConn>, Path(id): Path<String>) -> impl IntoResponse {
    let found = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let am: user::ActiveModel = user.into();
        am.delete(&db).await.unwrap();
        return (StatusCode::NO_CONTENT, Json::<Option<user::Model>>(None));
    }
    (StatusCode::NOT_FOUND, Json::<Option<user::Model>>(None))
}
