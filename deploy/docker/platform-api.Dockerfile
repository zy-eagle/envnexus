FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o platform-api ./services/platform-api/cmd/platform-api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/platform-api .
COPY --from=builder /app/services/platform-api/config ./config
EXPOSE 8080
CMD ["./platform-api"]
