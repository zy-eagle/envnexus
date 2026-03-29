# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY services/platform-api/go.mod services/platform-api/go.sum ./
ENV GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/platform-api/ .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o platform-api ./cmd/platform-api

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/platform-api .
COPY --from=builder /app/config ./config
EXPOSE 8080
CMD ["./platform-api"]
