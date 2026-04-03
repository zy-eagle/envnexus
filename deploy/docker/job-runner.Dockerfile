# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY services/job-runner/go.mod services/job-runner/go.sum ./
ENV GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY services/job-runner/ .
ARG ENX_BUILD_REVISION=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.buildRevision=${ENX_BUILD_REVISION}" -o job-runner ./cmd/job-runner

FROM alpine:latest
ARG ENX_BUILD_REVISION=unknown
LABEL org.opencontainers.image.revision="${ENX_BUILD_REVISION}"
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/job-runner .
COPY --from=builder /app/config ./config
EXPOSE 8082
CMD ["./job-runner"]
