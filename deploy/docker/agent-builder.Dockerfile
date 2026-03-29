# syntax=docker/dockerfile:1
# ══════════════════════════════════════════════════════════════════════════════
# Agent Builder — Fast rebuild using pre-built Electron shell
#
# Architecture:
#   Stage 1: Cross-compile Agent Core (Go) — always runs, ~30s
#   Stage 2: Package installers using pre-built electron shell — ~20s
#            (no npm install, no electron download, no network needed)
#   Stage 3: Upload to MinIO
#
# Prerequisites:
#   The enx-electron-shell image must exist locally. Build it once with:
#     docker build -f deploy/docker/electron-shell.Dockerfile -t enx-electron-shell .
#   Only rebuild when apps/agent-desktop/ source changes.
# ══════════════════════════════════════════════════════════════════════════════

# ── Stage 1: Cross-compile Agent Core (Go) ──────────────────────────────────
FROM golang:1.25-alpine AS go-builder

WORKDIR /app

COPY apps/agent-core/go.mod apps/agent-core/go.sum ./
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY apps/agent-core/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && \
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

# ── Stage 2: Build installers using pre-built Electron shell ─────────────────
# enx-electron-shell already has: node_modules, compiled TS, electron-builder cache
FROM enx-electron-shell AS desktop-builder

WORKDIR /project

# Only thing needed: fresh Agent Core binaries from Stage 1
COPY --from=go-builder /out/ ./bin/
RUN cp bin/enx-agent-linux-amd64 bin/enx-agent && \
    cp bin/enx-agent-windows-amd64.exe bin/enx-agent.exe && \
    chmod +x bin/enx-agent

# Build installers — FAST because everything is pre-cached:
#   - node_modules: already installed
#   - TypeScript: already compiled
#   - electron binary: already downloaded
#   - NSIS/wine tools: already cached
WORKDIR /project/apps/agent-desktop
RUN echo "=== Building Windows NSIS installer ===" && \
    npx electron-builder --win nsis --x64 \
        --config.directories.output=/installers/win-nsis \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent.exe \
    2>&1 | tail -15 && \
    echo "=== Building Windows Portable ZIP ===" && \
    npx electron-builder --win zip --x64 \
        --config.directories.output=/installers/win-zip \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent.exe \
    2>&1 | tail -15 && \
    echo "=== Injecting .portable marker into ZIP ===" && \
    echo "portable" > /tmp/.portable && \
    for f in /installers/win-zip/*.zip; do \
        if [ -f "$f" ]; then \
            (cd /tmp && zip -g "$f" .portable) && \
            echo "  ✓ Injected .portable into $(basename $f)"; \
        fi; \
    done && \
    echo "=== Building Linux AppImage ===" && \
    npx electron-builder --linux --x64 \
        --config.directories.output=/installers/linux \
        --config.extraResources.0.from=../../bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent \
    2>&1 | tail -15 && \
    echo "=== Normalizing output filenames ===" && \
    mkdir -p /out/installers && \
    win_found=false && \
    for f in /installers/win-nsis/*.exe; do \
        if [ -f "$f" ]; then cp "$f" /out/installers/EnvNexus-Agent-Setup-windows-amd64.exe && win_found=true && break; fi; \
    done && \
    if [ "$win_found" = "false" ]; then echo "ERROR: Windows NSIS installer was NOT produced!" && exit 1; fi && \
    for f in /installers/win-zip/*.zip; do \
        [ -f "$f" ] && cp "$f" /out/installers/EnvNexus-Agent-Portable-windows-amd64.zip && break; \
    done; \
    for f in /installers/linux/*.AppImage; do \
        [ -f "$f" ] && cp "$f" /out/installers/EnvNexus-Agent-linux-amd64.AppImage && break; \
    done; \
    echo "=== Produced installers ===" && \
    ls -lh /out/installers/

# ── Stage 3: Upload to MinIO ────────────────────────────────────────────────
FROM minio/mc:latest

COPY --from=desktop-builder /out/installers/ /installers/
COPY --from=go-builder /out/ /binaries/

COPY deploy/docker/upload-agents.sh /upload-agents.sh
RUN chmod +x /upload-agents.sh
ENTRYPOINT ["/upload-agents.sh"]
