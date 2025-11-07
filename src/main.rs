use axum::Router;
use dotenvy::dotenv;
use std::net::SocketAddr;

mod db;
mod models;
mod routes;
mod utils;

#[tokio::main]
async fn main() {
    dotenv().ok();
    let db = db::connect().await.expect("DB connection failed");

    let app = Router::new()
        .merge(routes::users::routes())
        .merge(routes::roles::routes())
        .merge(routes::apps::routes())
        .merge(routes::sessions::routes())
        .with_state(db);

    let addr = SocketAddr::from(([0, 0, 0, 0], 8001));
    println!("UniQUE API running at http://{}", addr);
    axum::serve(tokio::net::TcpListener::bind(addr).await.unwrap(), app)
        .await
        .unwrap();
}
