use axum::{
    Json, Router,
    extract::{Query, State},
    routing::get,
};
use sea_orm::*;
use serde::Deserialize;

use crate::models::role::{self, Entity as Role};

pub fn routes() -> Router<DbConn> {
    Router::new().route("/roles/search", get(search_roles))
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

pub async fn search_roles(
    State(db): State<crate::db::DbConn>,
    Query(params): Query<Pagination>,
) -> Json<serde_json::Value> {
    let (page, per_page, q, filter) = params.params();
    let page_index = page.saturating_sub(1);

    // 検索条件をためる Condition
    let mut cond = Condition::all();

    // q 検索（部分一致：名前-カスタムID）
    if let Some(ref kw) = q {
        let kw = kw.trim();
        if !kw.is_empty() {
            let like_kw = format!("%{}%", kw);
            cond = cond.add(
                Condition::any()
                    .add(role::Column::Name.like(&like_kw))
                    .add(role::Column::CustomId.like(&like_kw)),
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

                    match key.as_str() {
                        // 文字列系
                        "name" => cond = cond.add(role::Column::Name.contains(value)),
                        "custom_id" => cond = cond.add(role::Column::CustomId.contains(value)),

                        _ => {}
                    }
                }
            }
        }
    }

    // 条件を反映
    let select = Role::find().filter(cond).order_by_asc(role::Column::Name);

    // ページング
    let paginator = select.paginate(&db, per_page as u64);
    let users = paginator
        .fetch_page(page_index as u64)
        .await
        .unwrap_or_default();
    let total = paginator.num_items().await.unwrap_or(0);

    Json(serde_json::json!({
        "data": users,
        "meta": {
            "page": page,
            "per_page": per_page,
            "total": total
        }
    }))
}
