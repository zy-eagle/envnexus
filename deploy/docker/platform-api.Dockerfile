FROM golang:1.25-alpine AS builder
WORKDIR /app

# Layer 1: dependencies only (cached unless go.mod/go.sum change)
COPY services/platform-api/go.mod services/platform-api/go.sum ./
ENV GOWORK=off GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go mod download

# Layer 2: service source only
COPY services/platform-api/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o platform-api ./cmd/platform-api

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/platform-api .
COPY --from=builder /app/config ./config
EXPOSE 8080
CMD ["./platform-api"]
