FROM rust:1-alpine AS builder

WORKDIR /usr/src/app

COPY . .

RUN cargo install --path .

RUN cargo build --release

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /usr/src/app/target/release/UniQUE-API .

CMD ["./UniQUE-API"]