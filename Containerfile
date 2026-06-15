# Build stage
FROM rust:1.85 as builder

WORKDIR /app
COPY . .
RUN cargo build --release

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /app/target/release/linkshortener .
COPY templates/ templates/

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8081

ENV DATABASE_URL=sqlite:/data/links.db
ENV LISTEN_ADDR=0.0.0.0:8081

CMD ["./linkshortener"]
