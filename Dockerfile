FROM rust:1.82 AS builder
WORKDIR /app

COPY CHANGELOG.md ./CHANGELOG.md
COPY bot/ ./bot/

WORKDIR /app/bot

ENV SQLX_OFFLINE=true
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/bot/target/release/bot ./bot

ENTRYPOINT ["/app/bot"]
