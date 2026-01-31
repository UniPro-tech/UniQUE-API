FROM rust:1-alpine AS builder

RUN apk add --no-cache musl-dev mold clang build-base

WORKDIR /usr/src/app

COPY .cargo .cargo
COPY Cargo.toml Cargo.lock ./

RUN mkdir src && echo "fn main() {}" > src/main.rs

RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/usr/src/app/target \
    cargo build --release -j $(nproc)

RUN rm -rf src

COPY . .
RUN touch src/main.rs

RUN --mount=type=cache,target=/root/.cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/usr/src/app/target \
    cargo build --release -j $(nproc) && \
    cp target/release/UniQUE-API /usr/local/bin/

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /usr/local/bin/UniQUE-API .
CMD ["./UniQUE-API"]