FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o session-gateway ./services/session-gateway/cmd/session-gateway

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/session-gateway .
COPY --from=builder /app/services/session-gateway/config ./config
EXPOSE 8081
CMD ["./session-gateway"]
