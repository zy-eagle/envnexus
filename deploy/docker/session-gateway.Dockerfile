# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY services/session-gateway/go.mod services/session-gateway/go.sum ./
ENV GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/session-gateway/ .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o session-gateway ./cmd/session-gateway

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/session-gateway .
COPY --from=builder /app/config ./config
EXPOSE 8081
CMD ["./session-gateway"]
