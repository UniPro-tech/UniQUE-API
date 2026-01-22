use axum::{Json, Router, response::Html, routing::get};
use dotenvy::dotenv;
use std::net::SocketAddr;
use utoipa::OpenApi;

mod constants;
mod db;
mod docs;
mod middleware;
mod models;
mod routes;
mod utils;

#[tokio::main]
async fn main() {
    dotenv().ok();
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::DEBUG)
        .with_test_writer()
        .init();
    let db = db::connect().await.expect("DB connection failed");

    // configure CORS
    let app = Router::new()
        .route("/api-docs/openapi.json", get(openapi_json))
        .route("/swagger-ui", get(swagger_ui_html))
        .merge(routes::users::routes())
        .merge(routes::roles::routes())
        .merge(routes::apps::routes())
        .merge(routes::sessions::routes())
        .merge(routes::email_verify::routes())
        // apply simple CORS middleware (handles preflight and adds headers)
        .layer(axum::middleware::from_fn(cors_middleware))
        .layer(axum::middleware::from_fn_with_state(
            db.clone(),
            middleware::auth::auth_middleware,
        ))
        .with_state(db);

    let addr = SocketAddr::from(([0, 0, 0, 0], 8001));
    println!("UniQUE API running at http://{}", addr);
    println!("Swagger UI available at http://{}/swagger-ui", addr);
    axum::serve(tokio::net::TcpListener::bind(addr).await.unwrap(), app)
        .await
        .unwrap();
}

async fn openapi_json() -> Json<utoipa::openapi::OpenApi> {
    Json(docs::ApiDoc::openapi())
}

async fn swagger_ui_html() -> Html<&'static str> {
    Html(
        r#"
<!DOCTYPE html>
<html lang="ja">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="SwaggerUI" />
    <title>UniQUE API - Swagger UI</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: '/api-docs/openapi.json',
          dom_id: '#swagger-ui',
        });
      };
    </script>
  </body>
</html>
    "#,
    )
}

async fn cors_middleware(
    req: axum::http::Request<axum::body::Body>,
    next: axum::middleware::Next,
) -> impl axum::response::IntoResponse {
    use axum::http::{HeaderName, HeaderValue};

    if req.method() == &axum::http::Method::OPTIONS {
        let mut res = axum::http::Response::new(axum::body::Body::empty());
        let headers = res.headers_mut();
        headers.insert(
            HeaderName::from_static("access-control-allow-origin"),
            HeaderValue::from_static("*"),
        );
        headers.insert(
            HeaderName::from_static("access-control-allow-headers"),
            HeaderValue::from_static("*"),
        );
        headers.insert(
            HeaderName::from_static("access-control-allow-methods"),
            HeaderValue::from_static("GET,POST,PUT,DELETE,OPTIONS"),
        );
        return res;
    }

    let mut res = next.run(req).await;
    let headers = res.headers_mut();
    headers.insert(
        HeaderName::from_static("access-control-allow-origin"),
        HeaderValue::from_static("*"),
    );
    headers.insert(
        HeaderName::from_static("access-control-allow-headers"),
        HeaderValue::from_static("*"),
    );
    headers.insert(
        HeaderName::from_static("access-control-allow-methods"),
        HeaderValue::from_static("GET,POST,PUT,DELETE,OPTIONS"),
    );
    res
}
