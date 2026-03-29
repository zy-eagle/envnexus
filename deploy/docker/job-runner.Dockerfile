FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY services/job-runner/go.mod services/job-runner/go.sum ./
RUN go env -w GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

COPY services/job-runner/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o job-runner ./cmd/job-runner

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/job-runner .
COPY --from=builder /app/config ./config
EXPOSE 8082
CMD ["./job-runner"]
