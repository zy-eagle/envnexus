FROM golang:1.25-alpine AS builder

RUN apk --no-cache add curl

WORKDIR /app

COPY go.work go.work.sum ./
COPY apps/agent-core/go.mod apps/agent-core/go.sum ./apps/agent-core/
COPY services/platform-api/go.mod services/platform-api/go.sum ./services/platform-api/
COPY services/session-gateway/go.mod services/session-gateway/go.sum ./services/session-gateway/
COPY services/job-runner/go.mod services/job-runner/go.sum ./services/job-runner/
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

COPY . .

RUN mkdir -p /out && \
    for os in linux windows darwin; do \
        for arch in amd64 arm64; do \
            ext=""; \
            if [ "$os" = "windows" ]; then ext=".exe"; fi; \
            name="enx-agent-${os}-${arch}${ext}"; \
            echo "Building ${name}..."; \
            CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} \
                go build -ldflags="-s -w" -o /out/${name} ./apps/agent-core/cmd/enx-agent; \
            echo "  ✓ ${name} ($(du -h /out/${name} | cut -f1))"; \
        done; \
    done

FROM minio/mc:latest
COPY --from=builder /out/ /agents/
COPY deploy/docker/upload-agents.sh /upload-agents.sh
RUN chmod +x /upload-agents.sh
ENTRYPOINT ["/upload-agents.sh"]
