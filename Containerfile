# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /linkshortener ./cmd/linkshortener

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /linkshortener .
COPY templates/ templates/
RUN mkdir -p /data
EXPOSE 8081
ENV DATABASE_URL=/data/links.db
ENV LISTEN_ADDR=0.0.0.0:8081
CMD ["./linkshortener"]
