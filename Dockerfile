FROM rust:1-alpine AS builder

WORKDIR /usr/src/app

COPY . .

RUN apk add --no-cache musl-dev mold clang build-base

RUN cargo fix --bin "UniQUE-API"

RUN cargo build --release

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /usr/src/app/target/release/UniQUE-API .

CMD ["./UniQUE-API"]