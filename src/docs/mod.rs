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
        
        // Roles endpoints
        crate::routes::roles::get_all_roles,
        crate::routes::roles::get_role,
        crate::routes::roles::create_role,
        crate::routes::roles::patch_update_role,
        crate::routes::roles::delete_role,
        crate::routes::roles::put_role,
        
        // Roles sub-routes: Search
        crate::routes::roles_sub::search::search_roles,
        
        // Apps endpoints
        crate::routes::apps::get_all_apps,
        crate::routes::apps::get_app,
        crate::routes::apps::create_app,
        crate::routes::apps::patch_update_app,
        crate::routes::apps::delete_app,
        crate::routes::apps::put_app,
        
        // Sessions endpoints
        crate::routes::sessions::get_all_sessions,
        crate::routes::sessions::get_session,
        crate::routes::sessions::delete_session,
        
        // Email Verify endpoints (top-level)
        crate::routes::email_verify::get_email_verifications,
        crate::routes::email_verify::delete_email_verification,
        
        // Users sub-routes: Email Verify
        crate::routes::users_sub::email_verify::get_email_verifications,
        crate::routes::users_sub::email_verify::post_challenge,
        crate::routes::users_sub::email_verify::delete_email_verification,
        
        // Users sub-routes: Password
        crate::routes::users_sub::password::password_change,
        crate::routes::users_sub::password::password_reset,
        
        // Users sub-routes: Permissions
        crate::routes::users_sub::permissions::get_permissions_bit,
        
        // Users sub-routes: Roles
        crate::routes::users_sub::roles::get_all_roles,
        crate::routes::users_sub::roles::put_role,
        crate::routes::users_sub::roles::delete_role,
        
        // Users sub-routes: Search
        crate::routes::users_sub::search::search_users,
        
        // Users sub-routes: Sessions
        crate::routes::users_sub::sessions::get_all_sessions,
        crate::routes::users_sub::sessions::get_session,
        crate::routes::users_sub::sessions::delete_session,
        
        // Users sub-routes: Discord
        crate::routes::users_sub::discord::get_all_discord,
        crate::routes::users_sub::discord::put_discord,
        crate::routes::users_sub::discord::delete_discord,
    ),
    components(
        schemas(
            // Users
            crate::routes::users::PublicUserResponse,
            crate::routes::users::DetailedUserResponse,
            crate::routes::users::UserListResponse,
            crate::routes::users::UserResponse,
            crate::routes::users::CreateUser,
            crate::routes::users::PutUser,
            crate::routes::users::UpdateUser,
            
            // Roles
            crate::routes::roles::RoleResponse,
            crate::routes::roles::CreateRole,
            crate::routes::roles::UpdateRole,
            
            // Roles sub: Search
            crate::routes::roles_sub::search::SearchRolesResponse,
            crate::routes::roles_sub::search::SearchMetadata,
            
            // Apps
            crate::routes::apps::AppResponse,
            crate::routes::apps::GetAllAppsQuery,
            crate::routes::apps::CreateApp,
            crate::routes::apps::UpdateApp,
            
            // Sessions
            crate::routes::sessions::SessionResponse,
            
            // Email Verify
            crate::routes::email_verify::EmailVerificationResponse,
            
            // Users sub: Email Verify
            crate::routes::users_sub::email_verify::EmailVerificationResponse,
            crate::routes::users_sub::email_verify::CreateVerifyChallenge,
            
            // Users sub: Password
            crate::routes::users_sub::password::PasswordChange,
            crate::routes::users_sub::password::PasswordReset,
            
            // Users sub: Permissions
            crate::routes::users_sub::permissions::PermissionsResponse,
            
            // Users sub: Search
            crate::routes::users_sub::search::SearchUsersResponse,
            crate::routes::users_sub::search::SearchMetadata,
            
            // Users sub: Discord
            crate::routes::users_sub::discord::DiscordResponse,
            crate::routes::users_sub::discord::CreateDiscord,
        )
    ),
    tags(
        (name = "users", description = "ユーザー管理エンドポイント"),
        (name = "roles", description = "ロール管理エンドポイント"),
        (name = "apps", description = "アプリケーション管理エンドポイント"),
        (name = "sessions", description = "セッション管理エンドポイント"),
        (name = "email_verify", description = "Email検証エンドポイント"),
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
