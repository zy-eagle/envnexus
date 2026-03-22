FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct && CGO_ENABLED=0 GOOS=linux go build -o platform-api ./services/platform-api/cmd/platform-api

FROM alpine:latest
RUN apk --no-cache add curl tzdata
WORKDIR /app
COPY --from=builder /app/platform-api .
COPY --from=builder /app/services/platform-api/config ./config
EXPOSE 8080
CMD ["./platform-api"]
