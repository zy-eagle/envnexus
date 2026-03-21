FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o job-runner ./services/job-runner/cmd/job-runner

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/job-runner .
COPY --from=builder /app/services/job-runner/config ./config
EXPOSE 8082
CMD ["./job-runner"]
