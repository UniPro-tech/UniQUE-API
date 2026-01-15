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

use crate::routes::users_sub::discord::DiscordResponse;
use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::{
        discord,
        user::{self, Entity as User},
    },
    routes::{common_dtos::array_dto::ApiResponse, users_sub},
    utils::password,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

/// 公開用ユーザー情報(権限なしでも見られる情報)
#[derive(Serialize)]
pub struct PublicUserResponse {
    pub id: String,
    pub custom_id: String,
    pub name: String,
    pub email: String,
    pub period: Option<String>,
    pub is_enable: Option<bool>,
}

/// 詳細ユーザー情報(権限あり or 本人のみ)
#[derive(Serialize)]
pub struct DetailedUserResponse {
    pub id: String,
    pub custom_id: String,
    pub name: String,
    pub email: String,
    pub external_email: String,
    pub birthdate: Option<chrono::NaiveDate>,
    pub email_verified: bool,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
    pub created_at: Option<chrono::NaiveDateTime>,
    pub updated_at: Option<chrono::NaiveDateTime>,
    pub discords: Vec<DiscordResponse>,
}

/// ユーザーリストのレスポンス型（権限に応じて異なる型を返す）
#[derive(Serialize)]
#[serde(untagged)]
pub enum UserListResponse {
    Detailed(ApiResponse<Vec<DetailedUserResponse>>),
    Public(ApiResponse<Vec<PublicUserResponse>>),
}

/// 単一ユーザーのレスポンス型（権限に応じて異なる型を返す）
#[derive(Serialize)]
#[serde(untagged)]
pub enum UserResponse {
    Detailed(DetailedUserResponse),
    Public(PublicUserResponse),
}

impl From<user::Model> for PublicUserResponse {
    fn from(user: user::Model) -> Self {
        Self {
            id: user.id,
            custom_id: user.custom_id,
            name: user.name,
            email: user.email,
            period: user.period,
            is_enable: user.is_enable,
        }
    }
}

impl From<user::Model> for DetailedUserResponse {
    fn from(user: user::Model) -> Self {
        Self {
            id: user.id,
            custom_id: user.custom_id,
            name: user.name,
            email: user.email,
            external_email: user.external_email,
            birthdate: user.birthdate,
            email_verified: user.email_verified,
            period: user.period,
            joined_at: user.joined_at,
            is_system: user.is_system,
            is_enable: user.is_enable,
            is_suspended: user.is_suspended,
            suspended_until: user.suspended_until,
            suspended_reason: user.suspended_reason,
            created_at: user.created_at,
            updated_at: user.updated_at,
            discords: Vec::new(),
        }
    }
}

pub fn routes() -> Router<DbConn> {
    Router::new()
        .route("/users", get(get_all_users).post(create_user))
        .route(
            "/users/{id}",
            get(get_user)
                .patch(patch_update_user)
                .delete(delete_user)
                .put(put_user),
        )
        .merge(users_sub::discord::routes())
        .merge(users_sub::roles::routes())
        .merge(users_sub::password::routes())
        .merge(users_sub::search::routes())
        .merge(users_sub::sessions::routes())
        .merge(users_sub::email_verify::routes())
        .merge(users_sub::permissions::routes())
}

/// すべてのユーザーを取得するための関数
async fn get_all_users(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // USER_READ があるかどうかを判定（失敗しても許可し、後続でマスキング＆フィルタ）
    let has_user_read =
        permission_check::require_permission(&auth_user, Permission::USER_READ, &db)
            .await
            .is_ok();

    let users = User::find().all(&db).await.unwrap();

    if has_user_read {
        // 権限ありの場合は詳細情報を返す
        let detailed_users: Vec<DetailedUserResponse> =
            users.into_iter().map(DetailedUserResponse::from).collect();
        Ok((
            StatusCode::OK,
            Json(UserListResponse::Detailed(ApiResponse {
                data: detailed_users,
            })),
        ))
    } else {
        // 権限なしの場合は公開情報のみを返す
        // is_suspended=true または is_enable=false または tmp_ メールのユーザーを除外
        let public_users: Vec<PublicUserResponse> = users
            .into_iter()
            .filter(|u| {
                let suspended = u.is_suspended.unwrap_or(false);
                let enabled = u.is_enable.unwrap_or(true);
                let has_tmp_email = u.email.contains("tmp_");
                !suspended && enabled && !has_tmp_email
            })
            .map(PublicUserResponse::from)
            .collect();
        Ok((
            StatusCode::OK,
            Json(UserListResponse::Public(ApiResponse { data: public_users })),
        ))
    }
}

/// 特定のユーザーを取得するための関数
async fn get_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    // 自分自身かどうか & USER_READ 権限の有無をチェック
    let is_self = auth_user.user_id == id;
    let has_user_read =
        permission_check::require_permission(&auth_user, Permission::USER_READ, &db)
            .await
            .is_ok();

    let user = User::find_by_id(id.clone())
        .find_with_related(discord::Entity)
        .all(&db)
        .await
        .unwrap();

    if let Some((user_model, discord_models)) = user.into_iter().next() {
        // 権限が無く本人でもない場合は、is_suspended=true または is_enable=false のユーザーは非表示
        if !has_user_read && !is_self {
            let suspended = user_model.is_suspended.unwrap_or(false);
            let enabled = user_model.is_enable.unwrap_or(true);
            let has_tmp_email = user_model.email.contains("tmp_");
            if suspended || !enabled || has_tmp_email {
                return Err(StatusCode::NOT_FOUND);
            }
        }

        // 自分自身または権限ありの場合は詳細情報を返す
        if is_self || has_user_read {
            let mut detailed = DetailedUserResponse::from(user_model);
            detailed.discords = discord_models
                .into_iter()
                .map(DiscordResponse::from)
                .collect();
            Ok((StatusCode::OK, Json(UserResponse::Detailed(detailed))))
        } else {
            // それ以外は公開情報のみ
            let public = PublicUserResponse::from(user_model);
            Ok((StatusCode::OK, Json(UserResponse::Public(public))))
        }
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

#[derive(serde::Deserialize)]
struct CreateUser {
    pub custom_id: String,
    pub name: String,
    pub password: String,
    pub email: Option<String>,
    pub external_email: String,
    pub birthdate: Option<chrono::NaiveDate>,
    pub email_verified: Option<bool>,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
}

/// 新しいユーザーを作成するための関数
async fn create_user(
    State(db): State<DbConn>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<CreateUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::USER_CREATE, &db).await?;

    let password_hash = password::hash_password(&payload.password);
    let mut email = payload.email;
    if email.is_none() {
        if payload.period.is_some() {
            email = Some(format!(
                "{}.{}@uniproject.jp",
                payload.period.as_ref().unwrap(),
                payload.custom_id
            ));
        } else {
            email = Some(format!("temp_{}@uniproject.jp", payload.custom_id));
        }
    }

    let am = user::ActiveModel {
        id: Set(Ulid::new().to_string()),
        custom_id: Set(payload.custom_id),
        name: Set(payload.name),
        password_hash: Set(Some(password_hash)),
        email: Set(email.unwrap()),
        external_email: Set(payload.external_email),
        birthdate: Set(payload.birthdate),
        email_verified: Set(payload.email_verified.unwrap_or(false)),
        period: Set(payload.period),
        joined_at: Set(payload.joined_at),
        is_system: Set(Some(payload.is_system.unwrap_or(false))),
        created_at: Set(Some(Utc::now().naive_utc())),
        updated_at: Set(Some(Utc::now().naive_utc())),
        is_enable: Set(Some(payload.is_enable.unwrap_or(false))),
        is_suspended: Set(Some(payload.is_suspended.unwrap_or(false))),
        suspended_until: Set(payload.suspended_until),
        suspended_reason: Set(payload.suspended_reason),
    };
    let res = am.insert(&db).await.unwrap();
    Ok((StatusCode::CREATED, Json(res)))
}

#[derive(serde::Deserialize)]
struct PutUser {
    pub custom_id: String,
    pub name: String,
    pub password: Option<String>,
    pub email: Option<String>,
    pub external_email: String,
    pub birthdate: Option<chrono::NaiveDate>,
    pub email_verified: Option<bool>,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
}

async fn put_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<PutUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::USER_UPDATE, &id, &db)
        .await?;
    let password_hash = if let Some(password) = payload.password {
        password::hash_password(&password)
    } else {
        // 既存のパスワードハッシュを保持する
        let found = user::Entity::find_by_id(id.clone()).one(&db).await.unwrap();
        if let Some(user) = found {
            user.password_hash.unwrap_or_default()
        } else {
            "".to_string()
        }
    };
    let mut email = payload.email;
    if email.is_none() {
        if payload.period.is_some() {
            email = Some(format!(
                "{}.{}@uniproject.jp",
                payload.period.as_ref().unwrap(),
                payload.custom_id
            ));
        } else {
            email = Some(format!("temp_{}@uniproject.jp", payload.custom_id));
        }
    }

    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        am.name = Set(payload.name);
        am.email = Set(email.unwrap());
        am.external_email = Set(payload.external_email);
        am.birthdate = Set(payload.birthdate);
        am.email_verified = Set(payload.email_verified.unwrap_or(false));
        am.period = Set(payload.period);
        am.password_hash = Set(Some(password_hash));
        am.joined_at = Set(payload.joined_at);
        am.is_system = Set(Some(payload.is_system.unwrap_or(false)));
        am.is_enable = Set(Some(payload.is_enable.unwrap_or(false)));
        am.updated_at = Set(Some(Utc::now().naive_utc()));
        am.suspended_until = Set(payload.suspended_until);
        am.suspended_reason = Set(payload.suspended_reason);
        am.is_suspended = Set(Some(payload.is_suspended.unwrap_or(false)));
        am.joined_at = Set(payload.joined_at);
        let res = am.update(&db).await.unwrap();
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

#[derive(serde::Deserialize)]
struct UpdateUser {
    pub custom_id: Option<String>,
    pub name: Option<String>,
    pub password_hash: Option<String>,
    pub external_email: Option<String>,
    pub birthdate: Option<chrono::NaiveDate>,
    pub email_verified: Option<bool>,
    pub period: Option<String>,
    pub joined_at: Option<chrono::NaiveDateTime>,
    pub is_system: Option<bool>,
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub suspended_until: Option<chrono::NaiveDateTime>,
    pub suspended_reason: Option<String>,
    pub email: Option<String>,
}

/// ユーザーを差分アップデートするための関数
///
/// > このエンドポイントはOAuthの**アクセストークンでアクセス可能**です。
/// > ただし、システムのユーザーではない場合は、以下のフィールドのみ書き換え可能です。
/// > - name
/// > - external_email
async fn patch_update_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
    Json(payload): Json<UpdateUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission_or_self(&auth_user, Permission::USER_UPDATE, &id, &db)
        .await?;
    let found = user::Entity::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let mut am: user::ActiveModel = user.into();
        if let Some(custom_id) = payload.custom_id {
            am.custom_id = Set(custom_id);
        }
        if let Some(name) = payload.name {
            am.name = Set(name);
        }
        if let Some(external_email) = payload.external_email {
            am.external_email = Set(external_email);
        }
        if let Some(birthdate) = payload.birthdate {
            am.birthdate = Set(Some(birthdate));
        }
        if let Some(email_verified) = payload.email_verified {
            am.email_verified = Set(email_verified);
        }
        if let Some(password_hash) = payload.password_hash {
            am.password_hash = Set(Some(password_hash));
        }
        if let Some(period) = payload.period {
            am.period = Set(Some(period));
        }
        if let Some(joined_at) = payload.joined_at {
            am.joined_at = Set(Some(joined_at));
        }
        if let Some(is_system) = payload.is_system {
            am.is_system = Set(Some(is_system));
        }
        if let Some(is_enable) = payload.is_enable {
            am.is_enable = Set(Some(is_enable));
        }
        if let Some(is_suspended) = payload.is_suspended {
            am.is_suspended = Set(Some(is_suspended));
        }
        if let Some(suspended_until) = payload.suspended_until {
            am.suspended_until = Set(Some(suspended_until));
        }
        if let Some(suspended_reason) = payload.suspended_reason {
            am.suspended_reason = Set(Some(suspended_reason));
        }
        if let Some(email) = payload.email {
            am.email = Set(email);
        }
        am.updated_at = Set(Some(Utc::now().naive_utc()));
        let res = am.update(&db).await.unwrap();
        return Ok((StatusCode::OK, Json(Some(res))));
    }
    Err(StatusCode::NOT_FOUND)
}

/// ユーザーを削除するための関数
/// > [!IMPORTANT]
/// > このエンドポイントはOAuthの**アクセストークンでアクセス不可**です
async fn delete_user(
    State(db): State<DbConn>,
    Path(id): Path<String>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::USER_DELETE, &db).await?;
    let found = User::find_by_id(id).one(&db).await.unwrap();
    if let Some(user) = found {
        let am: user::ActiveModel = user.into();
        am.delete(&db).await.unwrap();
        return Ok((StatusCode::NO_CONTENT, Json::<Option<user::Model>>(None)));
    }
    Err(StatusCode::NOT_FOUND)
}
