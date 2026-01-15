use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    paths(
        // Users endpoints
        crate::routes::users::get_all_users,
        crate::routes::users::get_user,
        crate::routes::users::create_user,
        crate::routes::users::patch_update_user,
        crate::routes::users::delete_user,
        crate::routes::users::put_user,
    ),
    components(
        schemas(
            crate::routes::users::PublicUserResponse,
            crate::routes::users::DetailedUserResponse,
            crate::routes::users::UserListResponse,
            crate::routes::users::UserResponse,
        )
    ),
    tags(
        (name = "users", description = "ユーザー管理エンドポイント"),
    ),
    info(
        title = "UniQUE API",
        version = "1.0.0",
        description = "UniQUE認証システムのREST API",
        contact(
            name = "UniProject",
            url = "https://uniproject.jp"
        )
    )
)]
pub struct ApiDoc;
