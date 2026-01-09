use axum::{
    Json, Router,
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::get,
};
use chrono::{NaiveDate, NaiveDateTime, TimeZone, Utc};
use sea_orm::*;
use serde::Deserialize;

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::user::{self, Entity as User},
};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/users/search", get(search_users))
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct Pagination {
    pub page: Option<usize>,
    pub per_page: Option<usize>,
    pub q: Option<String>,
    pub filter: Option<String>,
}

impl Pagination {
    const DEFAULT_PAGE: usize = 1;
    const DEFAULT_PER_PAGE: usize = 10;

    pub fn params(&self) -> (usize, usize, Option<String>, Option<String>) {
        (
            self.page.unwrap_or(Self::DEFAULT_PAGE),
            self.per_page.unwrap_or(Self::DEFAULT_PER_PAGE),
            self.q.clone(),
            self.filter.clone(),
        )
    }
}

impl Default for Pagination {
    fn default() -> Self {
        Self {
            page: Some(Self::DEFAULT_PAGE),
            per_page: Some(Self::DEFAULT_PER_PAGE),
            q: None,
            filter: None,
        }
    }
}

pub async fn search_users(
    State(db): State<crate::db::DbConn>,
    Query(params): Query<Pagination>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::USER_READ, &db)
        .await?;

    let (page, per_page, q, filter) = params.params();
    let page_index = page.saturating_sub(1);

    // 検索条件をためる Condition
    let mut cond = Condition::all();

    // q 検索（部分一致：名前・メール・カスタムID）
    if let Some(ref kw) = q {
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

    // filter パラメータを解析
    if let Some(ref f) = filter {
        let f = f.trim();
        if !f.is_empty() {
            for pair in f.split(',') {
                if let Some((key, value)) = pair.split_once(':') {
                    let key = key.trim().to_lowercase();
                    let value = value.trim();

                    // 日付パース（YYYY-MM-DD or YYYY-MM-DD HH:MM:SS）
                    let parsed_date = NaiveDateTime::parse_from_str(value, "%Y-%m-%d %H:%M:%S")
                        .or_else(|_| {
                            NaiveDate::parse_from_str(value, "%Y-%m-%d")
                                .map(|d| d.and_hms_opt(0, 0, 0).unwrap())
                        })
                        .ok()
                        .map(|d| Utc.from_utc_datetime(&d));

                    match key.as_str() {
                        // bool系
                        "is_enable" => {
                            if let Ok(v) = value.parse::<bool>() {
                                cond = cond.add(user::Column::IsEnable.eq(v));
                            }
                        }
                        "is_suspended" => {
                            if let Ok(v) = value.parse::<bool>() {
                                cond = cond.add(user::Column::IsSuspended.eq(v));
                            }
                        }

                        // 文字列系
                        "name" => cond = cond.add(user::Column::Name.contains(value)),
                        "email" => cond = cond.add(user::Column::Email.contains(value)),
                        "external_email" => {
                            cond = cond.add(user::Column::ExternalEmail.contains(value))
                        }
                        "period" => cond = cond.add(user::Column::Period.contains(value)),

                        // 日付系（before / after）
                        "joined_before" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::JoinedAt.lt(date));
                            }
                        }
                        "joined_after" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::JoinedAt.gt(date));
                            }
                        }
                        "created_before" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::CreatedAt.lt(date));
                            }
                        }
                        "created_after" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::CreatedAt.gt(date));
                            }
                        }
                        "suspended_before" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::SuspendedUntil.lt(date));
                            }
                        }
                        "suspended_after" => {
                            if let Some(date) = parsed_date {
                                cond = cond.add(user::Column::SuspendedUntil.gt(date));
                            }
                        }

                        _ => {}
                    }
                }
            }
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

    Ok(Json(serde_json::json!({
        "data": users,
        "meta": {
            "page": page,
            "per_page": per_page,
            "total": total
        }
    })))
}
