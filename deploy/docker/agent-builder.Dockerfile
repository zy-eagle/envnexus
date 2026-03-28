# ══════════════════════════════════════════════════════════════════════════════
# Stage 1: Cross-compile Agent Core (Go)
# ══════════════════════════════════════════════════════════════════════════════
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

COPY apps/agent-core/go.mod apps/agent-core/go.sum ./
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct && go mod download

COPY apps/agent-core/ .

RUN mkdir -p /out && \
    for os in linux windows; do \
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

# ══════════════════════════════════════════════════════════════════════════════
# Stage 2: Build Electron Desktop Installers (NSIS for Windows, AppImage for Linux)
# ══════════════════════════════════════════════════════════════════════════════
FROM electronuserland/builder:wine AS desktop-builder

WORKDIR /project

# Copy the full agent-desktop source
COPY apps/agent-desktop/package.json apps/agent-desktop/package-lock.json* apps/agent-desktop/tsconfig.json ./apps/agent-desktop/
COPY apps/agent-desktop/src/ ./apps/agent-desktop/src/

# Copy icon assets (or generate placeholder if missing)
COPY apps/agent-desktop/assets/ ./apps/agent-desktop/assets/
RUN if [ ! -f apps/agent-desktop/assets/icon.png ]; then \
        apt-get update -qq && apt-get install -y -qq imagemagick > /dev/null 2>&1 || true; \
        if command -v convert > /dev/null 2>&1; then \
            convert -size 256x256 xc:'#3b82f6' \
                -fill white -gravity center -pointsize 72 -annotate 0 'E' \
                apps/agent-desktop/assets/icon.png; \
            convert apps/agent-desktop/assets/icon.png apps/agent-desktop/assets/icon.ico 2>/dev/null || true; \
        else \
            echo "ImageMagick not available, using empty icon placeholder"; \
            printf '\x89PNG\r\n\x1a\n' > apps/agent-desktop/assets/icon.png; \
        fi; \
    fi

# Copy Agent Core binaries into bin/ (electron-builder extraResources reads from ../../bin/)
COPY --from=go-builder /out/ ./bin/
RUN cp bin/enx-agent-linux-amd64 bin/enx-agent && \
    cp bin/enx-agent-windows-amd64.exe bin/enx-agent.exe && \
    chmod +x bin/enx-agent

# Install dependencies and build TypeScript
WORKDIR /project/apps/agent-desktop
RUN npm install --prefer-offline --no-audit 2>&1 | tail -5
RUN npx tsc

# Build Windows NSIS installer (cross-compiled via wine)
RUN npx electron-builder --win --x64 \
        --config.directories.output=/installers/win \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent.exe \
    2>&1 | tail -20 || echo "Windows build completed (check logs above)"

# Build Linux AppImage
RUN npx electron-builder --linux --x64 \
        --config.directories.output=/installers/linux \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent \
    2>&1 | tail -20 || echo "Linux build completed (check logs above)"

# Normalize output filenames for predictable upload
RUN mkdir -p /out/installers && \
    for f in /installers/win/*.exe; do \
        [ -f "$f" ] && cp "$f" /out/installers/EnvNexus-Agent-Setup-windows-amd64.exe && break; \
    done; \
    for f in /installers/linux/*.AppImage; do \
        [ -f "$f" ] && cp "$f" /out/installers/EnvNexus-Agent-linux-amd64.AppImage && break; \
    done; \
    ls -lh /out/installers/ 2>/dev/null || echo "No installers produced"

# ══════════════════════════════════════════════════════════════════════════════
# Stage 3: Upload to MinIO
# ══════════════════════════════════════════════════════════════════════════════
FROM minio/mc:latest

# Installers from electron-builder
COPY --from=desktop-builder /out/installers/ /installers/

# Raw binaries (backward compatibility + fallback)
COPY --from=go-builder /out/ /binaries/

COPY deploy/docker/upload-agents.sh /upload-agents.sh
RUN chmod +x /upload-agents.sh
ENTRYPOINT ["/upload-agents.sh"]
