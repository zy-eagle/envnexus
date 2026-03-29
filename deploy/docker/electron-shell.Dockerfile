# syntax=docker/dockerfile:1
# ══════════════════════════════════════════════════════════════════════════════
# Electron Shell — Pre-built base image for desktop installers
#
# This image contains:
#   - node_modules (npm dependencies)
#   - Compiled TypeScript (dist/)
#   - electron-builder cache (NSIS, wine, etc.)
#
# Build this ONCE (or when apps/agent-desktop/ source changes):
#   docker build -f deploy/docker/electron-shell.Dockerfile -t enx-electron-shell .
#
# Then agent-builder.Dockerfile uses this as a base — no more npm install,
# no more electron download, no more network failures.
# ══════════════════════════════════════════════════════════════════════════════
FROM electronuserland/builder:wine

RUN apt-get -qq update && apt-get -qq install -y zip >/dev/null 2>&1 && rm -rf /var/lib/apt/lists/*

WORKDIR /project/apps/agent-desktop

# Layer 1: Dependencies — cached until package.json changes
COPY apps/agent-desktop/package.json apps/agent-desktop/package-lock.json* ./

RUN --mount=type=cache,target=/root/.npm \
    ELECTRON_MIRROR=https://npmmirror.com/mirrors/electron/ \
    ELECTRON_BUILDER_BINARIES_MIRROR=https://npmmirror.com/mirrors/electron-builder-binaries/ \
    npm install --prefer-offline --no-audit --no-fund && \
    echo "✓ npm install complete"

# Layer 2: Source + assets + build scripts + TypeScript compile
COPY apps/agent-desktop/tsconfig.json ./
COPY apps/agent-desktop/src/ ./src/
COPY apps/agent-desktop/assets/ ./assets/
COPY apps/agent-desktop/build/ ./build/

RUN npx tsc && echo "✓ TypeScript compiled"

# Pre-warm electron-builder cache (download NSIS tools, wine config, etc.)
# Use a dummy binary so electron-builder downloads all platform tools.
RUN mkdir -p /project/bin && \
    echo "placeholder" > /project/bin/enx-agent && \
    echo "placeholder" > /project/bin/enx-agent.exe && \
    chmod +x /project/bin/enx-agent && \
    (npx electron-builder --win --x64 --dir \
        --config.directories.output=/tmp/warmup-win \
        --config.extraResources.0.from=/project/bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent.exe \
    2>&1 | tail -5 || true) && \
    (npx electron-builder --linux --x64 --dir \
        --config.directories.output=/tmp/warmup-linux \
        --config.extraResources.0.from=/project/bin/ \
        --config.extraResources.0.to=bin \
        --config.extraResources.0.filter[0]=enx-agent \
    2>&1 | tail -5 || true) && \
    rm -rf /tmp/warmup-* /project/bin && \
    echo "✓ electron-builder cache warmed"

# Mark as ready
RUN echo "enx-electron-shell ready" > /project/.shell-ready
