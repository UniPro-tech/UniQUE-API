use axum::{
    Json, Router,
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::get,
};
use chrono::{DateTime, NaiveDate, NaiveDateTime, TimeZone, Utc};
use sea_orm::*;
use serde::{Deserialize, Serialize};
use utoipa::{IntoParams, ToSchema};

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::user::{self, Entity as User},
    routes::users::PublicUserResponse,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize, ToSchema)]
pub struct SearchMetadata {
    pub page: usize,
    pub per_page: usize,
    pub total: u64,
    pub total_pages: u64,
}

#[derive(Serialize, ToSchema)]
pub struct SearchUsersResponse {
    pub data: Vec<crate::routes::users::PublicUserResponse>,
    pub meta: SearchMetadata,
}

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users/search", get(search_users))
}

#[derive(Debug, Deserialize, IntoParams, ToSchema)]
#[serde(default)]
pub struct SearchParams {
    pub page: Option<usize>,
    pub per_page: Option<usize>,
    pub q: Option<String>,
    // 個別フィルタ
    pub is_enable: Option<bool>,
    pub is_suspended: Option<bool>,
    pub name: Option<String>,
    pub email: Option<String>,
    pub external_email: Option<String>,
    pub period: Option<String>,
    pub joined_before: Option<String>,
    pub joined_after: Option<String>,
    pub created_before: Option<String>,
    pub created_after: Option<String>,
    pub suspended_before: Option<String>,
    pub suspended_after: Option<String>,
}

impl SearchParams {
    const DEFAULT_PAGE: usize = 1;
    const DEFAULT_PER_PAGE: usize = 10;

    pub fn page(&self) -> usize {
        self.page.unwrap_or(Self::DEFAULT_PAGE)
    }

    pub fn per_page(&self) -> usize {
        self.per_page.unwrap_or(Self::DEFAULT_PER_PAGE)
    }
}

impl Default for SearchParams {
    fn default() -> Self {
        Self {
            page: None,
            per_page: None,
            q: None,
            is_enable: None,
            is_suspended: None,
            name: None,
            email: None,
            external_email: None,
            period: None,
            joined_before: None,
            joined_after: None,
            created_before: None,
            created_after: None,
            suspended_before: None,
            suspended_after: None,
        }
    }
}

#[utoipa::path(
    get,
    path = "/users/search",
    tag = "users",
    params(SearchParams),
    responses(
        (status = 200, description = "ユーザー検索成功", body = SearchUsersResponse),
        (status = 403, description = "アクセス権限なし")
    ),
    security(
        ("session_token" = [])
    )
)]
pub async fn search_users(
    State(db): State<DbConn>,
    Query(params): Query<SearchParams>,
    axum::Extension(auth_user): axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::USER_READ, &db).await?;

    let page = params.page();
    let per_page = params.per_page();
    let page_index = page.saturating_sub(1);

    // 検索条件をためる Condition
    let mut cond = Condition::all();

    // q 検索（部分一致：名前・メール・カスタムID）
    if let Some(ref kw) = params.q {
        let kw = kw.trim();
        if !kw.is_empty() {
            let like_kw = format!("%{}%", kw);
            cond = cond.add(
                Condition::any()
                    .add(user::Column::Name.like(&like_kw))
                    .add(user::Column::Email.like(&like_kw))
                    .add(user::Column::CustomId.like(&like_kw)),
            );
        }
    }

    // 個別フィルタの適用
    if let Some(v) = params.is_enable {
        cond = cond.add(user::Column::IsEnable.eq(v));
    }

    if let Some(v) = params.is_suspended {
        cond = cond.add(user::Column::IsSuspended.eq(v));
    }

    if let Some(ref v) = params.name {
        cond = cond.add(user::Column::Name.contains(v));
    }

    if let Some(ref v) = params.email {
        cond = cond.add(user::Column::Email.contains(v));
    }

    if let Some(ref v) = params.external_email {
        cond = cond.add(user::Column::ExternalEmail.contains(v));
    }

    if let Some(ref v) = params.period {
        cond = cond.add(user::Column::Period.contains(v));
    }

    // ヘルパー関数：文字列から日時をパース
    let parse_datetime = |s: &str| -> Option<DateTime<Utc>> {
        NaiveDateTime::parse_from_str(s, "%Y-%m-%d %H:%M:%S")
            .or_else(|_| {
                NaiveDate::parse_from_str(s, "%Y-%m-%d").map(|d| d.and_hms_opt(0, 0, 0).unwrap())
            })
            .ok()
            .map(|d| Utc.from_utc_datetime(&d))
    };

    if let Some(ref v) = params.joined_before {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::JoinedAt.lt(date));
        }
    }

    if let Some(ref v) = params.joined_after {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::JoinedAt.gt(date));
        }
    }

    if let Some(ref v) = params.created_before {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::CreatedAt.lt(date));
        }
    }

    if let Some(ref v) = params.created_after {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::CreatedAt.gt(date));
        }
    }

    if let Some(ref v) = params.suspended_before {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::SuspendedUntil.lt(date));
        }
    }

    if let Some(ref v) = params.suspended_after {
        if let Some(date) = parse_datetime(v) {
            cond = cond.add(user::Column::SuspendedUntil.gt(date));
        }
    }

    // 条件を反映
    let select = User::find().filter(cond).order_by_asc(user::Column::Name);

    // ページング
    let paginator = select.paginate(&db, per_page as u64);
    let users = paginator
        .fetch_page(page_index as u64)
        .await
        .unwrap_or_default();
    let total = paginator.num_items().await.unwrap_or(0);

    let total_pages = (total as usize + per_page - 1) / per_page;
    let data: Vec<PublicUserResponse> = users.into_iter().map(PublicUserResponse::from).collect();

    Ok((
        StatusCode::OK,
        Json(SearchUsersResponse {
            data,
            meta: SearchMetadata {
                page,
                per_page,
                total,
                total_pages: total_pages as u64,
            },
        }),
    ))
}
