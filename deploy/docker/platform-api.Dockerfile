# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
# Monorepo layout: go.mod replace ../../libs/shared must resolve (from services/platform-api)
WORKDIR /src

COPY libs/shared ./libs/shared

COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/
WORKDIR /src/services/platform-api
ENV GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/platform-api/ .
ARG ENX_BUILD_REVISION=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.buildRevision=${ENX_BUILD_REVISION}" -o platform-api ./cmd/platform-api
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o enx-migrate ./cmd/migrate

FROM alpine:latest
ARG ENX_BUILD_REVISION=unknown
LABEL org.opencontainers.image.revision="${ENX_BUILD_REVISION}"
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /src/services/platform-api/platform-api .
COPY --from=builder /src/services/platform-api/enx-migrate .
COPY --from=builder /src/services/platform-api/config ./config
EXPOSE 8080
CMD ["./platform-api"]
