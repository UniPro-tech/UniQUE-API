
FROM rust:1-alpine AS builder

RUN apk add --no-cache musl-dev mold clang build-base

WORKDIR /usr/src/app

COPY .cargo .cargo

COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs

RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    cargo build --release -j $(nproc)

RUN rm -rf src

COPY . .
RUN touch src/main.rs

RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    cargo build --release -j $(nproc)

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /usr/src/app/target/release/UniQUE-API .
CMD ["./UniQUE-API"]