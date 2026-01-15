use axum::{
    Json, Router,
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::get,
};
use sea_orm::*;
use serde::{Deserialize, Serialize};

use crate::{
    constants::permissions::Permission,
    middleware::{auth::AuthUser, permission_check},
    models::role::{self, Entity as Role},
    routes::roles::RoleResponse,
};

/// =======================
/// DTO（レスポンス専用）
/// =======================

#[derive(Serialize)]
pub struct SearchMetadata {
    pub page: usize,
    pub per_page: usize,
    pub total: u64,
    pub total_pages: u64,
}

#[derive(Serialize)]
pub struct SearchRolesResponse {
    pub data: Vec<RoleResponse>,
    pub meta: SearchMetadata,
}

pub fn routes() -> Router<DbConn> {
    Router::new().route("/roles/search", get(search_roles))
}

#[derive(Debug, Deserialize)]
#[serde(default)]
pub struct SearchParams {
    pub page: Option<usize>,
    pub per_page: Option<usize>,
    pub q: Option<String>,
    pub name: Option<String>,
    pub custom_id: Option<String>,
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
            name: None,
            custom_id: None,
        }
    }
}

pub async fn search_roles(
    State(db): State<crate::db::DbConn>,
    Query(params): Query<SearchParams>,
    auth_user: axum::Extension<AuthUser>,
) -> Result<impl IntoResponse, StatusCode> {
    permission_check::require_permission(&auth_user, Permission::ROLE_MANAGE, &db).await?;

    let page = params.page();
    let per_page = params.per_page();
    let page_index = page.saturating_sub(1);

    // 検索条件をためる Condition
    let mut cond = Condition::all();

    // q 検索（部分一致：名前・カスタムID）
    if let Some(ref kw) = params.q {
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

    // 個別フィルタの適用
    if let Some(ref v) = params.name {
        cond = cond.add(role::Column::Name.contains(v));
    }

    if let Some(ref v) = params.custom_id {
        cond = cond.add(role::Column::CustomId.contains(v));
    }

    // 条件を反映
    let select = Role::find().filter(cond).order_by_asc(role::Column::Name);

    // ページング
    let paginator = select.paginate(&db, per_page as u64);
    let roles = paginator
        .fetch_page(page_index as u64)
        .await
        .unwrap_or_default();
    let total = paginator.num_items().await.unwrap_or(0);

    let total_pages = (total as usize + per_page - 1) / per_page;
    let data: Vec<RoleResponse> = roles.into_iter().map(RoleResponse::from).collect();

    Ok((
        StatusCode::OK,
        Json(SearchRolesResponse {
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
