FROM golang:1.25-alpine AS builder

WORKDIR /app

# Layer 1: Dependencies only (cached unless go.mod/go.sum change)
COPY apps/agent-core/go.mod apps/agent-core/go.sum ./
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

# Layer 2: Source code only (cached unless agent-core source changes)
# Other services are NOT copied — agent-core has no cross-module imports.
COPY apps/agent-core/ .

# Layer 3: Cross-compile all platforms (fully cached if layers above unchanged)
RUN mkdir -p /out && \
    for os in linux windows darwin; do \
        for arch in amd64 arm64; do \
            ext=""; \
            if [ "$os" = "windows" ]; then ext=".exe"; fi; \
            name="enx-agent-${os}-${arch}${ext}"; \
            echo "Building ${name}..."; \
            CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} \
                go build -ldflags="-s -w" -o /out/${name} ./cmd/enx-agent; \
            echo "  ✓ ${name} ($(du -h /out/${name} | cut -f1))"; \
        done; \
    done

# Runtime stage: upload to MinIO
FROM minio/mc:latest
COPY --from=builder /out/ /agents/
COPY deploy/docker/upload-agents.sh /upload-agents.sh
RUN chmod +x /upload-agents.sh
ENTRYPOINT ["/upload-agents.sh"]
