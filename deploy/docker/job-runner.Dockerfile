FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.work go.work.sum ./
COPY services/job-runner/go.mod services/job-runner/go.sum ./services/job-runner/
COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/
COPY services/session-gateway/go.mod services/session-gateway/go.sum ./services/session-gateway/
COPY apps/agent-core/go.mod apps/agent-core/go.sum ./apps/agent-core/
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o job-runner ./services/job-runner/cmd/job-runner

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/job-runner .
COPY --from=builder /app/services/job-runner/config ./config
EXPOSE 8082
CMD ["./job-runner"]
