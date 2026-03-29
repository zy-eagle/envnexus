# syntax=docker/dockerfile:1
# ══════════════════════════════════════════════════════════════════════════════
# Stage 1: Cross-compile Agent Core (Go)
# Cache: invalidated only when apps/agent-core/ source changes
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
# Stage 2: Build Electron Desktop Installers (GUI + Agent Core unified)
#
# Speed optimizations:
#   1. BuildKit cache mount for npm — survives across builds (~180MB electron
#      binary downloaded once, reused forever)
#   2. Icon assets committed to repo — no more apt-get imagemagick at build time
#   3. Win + Linux built in a single RUN — avoids double node_modules scan
#   4. Electron download mirror set to Chinese CDN
# ══════════════════════════════════════════════════════════════════════════════
FROM electronuserland/builder:wine AS desktop-builder

WORKDIR /project

# Layer 1: package.json only → npm install is cached until deps change
COPY apps/agent-desktop/package.json apps/agent-desktop/package-lock.json* ./apps/agent-desktop/

WORKDIR /project/apps/agent-desktop

# Use BuildKit cache mount so the npm cache + electron zip persist across builds.
# First build downloads ~180MB electron; subsequent builds skip entirely.
RUN --mount=type=cache,target=/root/.npm \
    --mount=type=cache,target=/root/.cache/electron \
    ELECTRON_MIRROR=https://npmmirror.com/mirrors/electron/ \
    ELECTRON_BUILDER_BINARIES_MIRROR=https://npmmirror.com/mirrors/electron-builder-binaries/ \
    npm install --prefer-offline --no-audit --no-fund 2>&1 | tail -10

WORKDIR /project

# Layer 2: source + config + assets (icon files committed, no imagemagick needed)
COPY apps/agent-desktop/tsconfig.json ./apps/agent-desktop/
COPY apps/agent-desktop/src/ ./apps/agent-desktop/src/
COPY apps/agent-desktop/assets/ ./apps/agent-desktop/assets/

# Layer 3: Agent Core binaries from Stage 1
COPY --from=go-builder /out/ ./bin/
RUN cp bin/enx-agent-linux-amd64 bin/enx-agent && \
    cp bin/enx-agent-windows-amd64.exe bin/enx-agent.exe && \
    chmod +x bin/enx-agent

# Layer 4: TypeScript compile + build ALL installers in one RUN
# Combining win+linux avoids re-scanning node_modules twice.
WORKDIR /project/apps/agent-desktop
RUN --mount=type=cache,target=/root/.cache/electron \
    --mount=type=cache,target=/root/.cache/electron-builder \
    npx tsc && \
    echo "=== Building Windows NSIS installer ===" && \
    npx electron-builder --win --x64 \
        --config.directories.output=/installers/win \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent.exe \
    2>&1 | tail -15 && \
    echo "=== Building Linux AppImage ===" && \
    npx electron-builder --linux --x64 \
        --config.directories.output=/installers/linux \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent \
    2>&1 | tail -15 && \
    echo "=== Normalizing output filenames ===" && \
    mkdir -p /out/installers && \
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

COPY --from=desktop-builder /out/installers/ /installers/
COPY --from=go-builder /out/ /binaries/

COPY deploy/docker/upload-agents.sh /upload-agents.sh
RUN chmod +x /upload-agents.sh
ENTRYPOINT ["/upload-agents.sh"]
