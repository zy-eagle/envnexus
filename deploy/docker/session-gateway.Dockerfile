FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct && CGO_ENABLED=0 GOOS=linux go build -o session-gateway ./services/session-gateway/cmd/session-gateway

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/session-gateway .
COPY --from=builder /app/services/session-gateway/config ./config
EXPOSE 8081
CMD ["./session-gateway"]
