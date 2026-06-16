# Build stage
FROM rust:1.96 as builder

RUN apt-get update && apt-get install -y musl-tools
RUN rustup target add x86_64-unknown-linux-musl

WORKDIR /app
COPY . .
RUN cargo build --release --target x86_64-unknown-linux-musl

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/linkshortener .
COPY templates/ templates/

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8081

ENV DATABASE_URL=sqlite:/data/links.db
ENV LISTEN_ADDR=0.0.0.0:8081

CMD ["./linkshortener"]
