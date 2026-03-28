FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.work go.work.sum ./
COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/
COPY apps/agent-core/go.mod apps/agent-core/go.sum ./apps/agent-core/
COPY services/session-gateway/go.mod services/session-gateway/go.sum ./services/session-gateway/
COPY services/job-runner/go.mod services/job-runner/go.sum ./services/job-runner/
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o platform-api ./services/platform-api/cmd/platform-api

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/platform-api .
COPY --from=builder /app/services/platform-api/config ./config
EXPOSE 8080
CMD ["./platform-api"]
