FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.work go.work.sum ./
COPY services/session-gateway/go.mod services/session-gateway/go.sum ./services/session-gateway/
COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/
COPY apps/agent-core/go.mod apps/agent-core/go.sum ./apps/agent-core/
COPY services/job-runner/go.mod services/job-runner/go.sum ./services/job-runner/
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o session-gateway ./services/session-gateway/cmd/session-gateway

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/session-gateway .
COPY --from=builder /app/services/session-gateway/config ./config
EXPOSE 8081
CMD ["./session-gateway"]
