#!/bin/bash
set -euo pipefail

# EnvNexus — Smart Deployment Script
# Detects which services actually changed and only rebuilds those.

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="${SCRIPT_DIR}/deploy/docker"
HASH_DIR="${DEPLOY_DIR}/.build-hashes"

# ── Helpers ──────────────────────────────────────────────────────────────────

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_skip()  { echo -e "${DIM}[SKIP]${NC}  $*"; }

gen_secret() {
    if command -v openssl &>/dev/null; then
        openssl rand -hex 16
    elif [ -f /dev/urandom ]; then
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    else
        date +%s%N | sha256sum | head -c 32
    fi
}

detect_host_ip() {
    local ip=""
    if command -v hostname &>/dev/null && hostname -I &>/dev/null 2>&1; then
        ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    if [ -z "$ip" ] && command -v ipconfig.exe &>/dev/null; then
        # WSL/Windows fallback
        ip=$(ipconfig.exe | grep -i "IPv4" | grep -v "127.0.0.1" | awk -F': ' '{print $2}' | tr -d '\r' | head -n 1 || true)
    fi
    if [ -z "$ip" ] && command -v ipconfig &>/dev/null; then
        # macOS fallback
        ip=$(ipconfig getifaddr en0 2>/dev/null || true)
    fi
    if [ -z "$ip" ]; then
        ip="127.0.0.1"
    fi
    echo "$ip"
}

check_env() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker not found. Please install Docker first: https://docs.docker.com/get-docker/"
        exit 1
    fi
    if ! docker compose version &>/dev/null; then
        log_error "Docker Compose V2 not found. Please upgrade Docker."
        exit 1
    fi
}

# ── Content hash for change detection ────────────────────────────────────────
# Computes a hash of the source files relevant to a service.
# If the hash matches the last successful build, the service is skipped.

mkdir -p "$HASH_DIR"

compute_hash() {
    local dir="$1"
    if [ ! -d "${SCRIPT_DIR}/${dir}" ]; then
        echo "missing"
        return
    fi

    # Fast path: use git if inside a git repo (much faster than find+sha256sum)
    if git -C "${SCRIPT_DIR}" rev-parse --is-inside-work-tree &>/dev/null; then
        local tree_hash diff_hash untracked_hash
        tree_hash=$(git -C "${SCRIPT_DIR}" ls-tree -r HEAD -- "$dir" 2>/dev/null | sha256sum | awk '{print $1}')
        diff_hash=$(git -C "${SCRIPT_DIR}" diff HEAD -- "$dir" 2>/dev/null | sha256sum | awk '{print $1}')
        untracked_hash=$(git -C "${SCRIPT_DIR}" ls-files --others --exclude-standard -- "$dir" 2>/dev/null | sha256sum | awk '{print $1}')
        echo "${tree_hash}${diff_hash}${untracked_hash}" | sha256sum | awk '{print $1}'
        return
    fi

    # Fallback: full filesystem scan (non-git environments)
    find "${SCRIPT_DIR}/${dir}" -type f \
        -not -path '*/node_modules/*' \
        -not -path '*/.next/*' \
        -not -path '*/dist/*' \
        -not -path '*/release/*' \
        -not -path '*/__pycache__/*' \
        -print0 2>/dev/null | sort -z | xargs -0 sha256sum 2>/dev/null | sha256sum | awk '{print $1}'
}

# Also hash workspace-level Go files that affect all Go services
compute_go_workspace_hash() {
    local files=""
    for f in go.work go.work.sum; do
        if [ -f "${SCRIPT_DIR}/${f}" ]; then
            files="${files} ${SCRIPT_DIR}/${f}"
        fi
    done
    if [ -n "$files" ]; then
        sha256sum $files 2>/dev/null | sha256sum | awk '{print $1}'
    else
        echo "none"
    fi
}

has_changed() {
    local service="$1"
    local current_hash="$2"
    local hash_file="${HASH_DIR}/${service}.hash"

    if [ ! -f "$hash_file" ]; then
        return 0  # No previous hash = needs build
    fi

    local prev_hash
    prev_hash=$(cat "$hash_file")
    if [ "$prev_hash" = "$current_hash" ]; then
        return 1  # Same hash = no change
    fi
    return 0  # Different hash = changed
}

save_hash() {
    local service="$1"
    local hash="$2"
    echo "$hash" > "${HASH_DIR}/${service}.hash"
}

# ── .env generation ──────────────────────────────────────────────────────────

patch_env() {
    local env_file="${DEPLOY_DIR}/.env"
    [ -f "$env_file" ] || return

    local HOST_IP
    HOST_IP=$(detect_host_ip)

    if ! grep -q "^ENX_OBJECT_STORAGE_PUBLIC_ENDPOINT=" "$env_file" 2>/dev/null; then
        log_info "Patching .env: adding ENX_OBJECT_STORAGE_PUBLIC_ENDPOINT=${HOST_IP}:9000"
        echo "" >> "$env_file"
        echo "# ===== Auto-patched: MinIO public endpoint for presigned URLs =====" >> "$env_file"
        echo "ENX_OBJECT_STORAGE_PUBLIC_ENDPOINT=${HOST_IP}:9000" >> "$env_file"
    fi

    # Fix PUBLIC URLs that still point to localhost — these are used by Agent Desktop
    # running on end-user machines, so they must be the server's real IP.
    if [ "$HOST_IP" != "127.0.0.1" ]; then
        if grep -q "^ENX_PLATFORM_API_PUBLIC_BASE_URL=http://localhost:" "$env_file" 2>/dev/null; then
            local port
            port=$(grep "^ENX_PLATFORM_API_PUBLIC_BASE_URL=" "$env_file" | sed 's/.*localhost://' | tr -d '\r')
            sed -i "s|^ENX_PLATFORM_API_PUBLIC_BASE_URL=http://localhost:.*|ENX_PLATFORM_API_PUBLIC_BASE_URL=http://${HOST_IP}:${port}|" "$env_file"
            log_info "Patched ENX_PLATFORM_API_PUBLIC_BASE_URL: localhost -> ${HOST_IP}"
        fi
        if grep -q "^ENX_SESSION_GATEWAY_PUBLIC_WS_URL=ws://localhost:" "$env_file" 2>/dev/null; then
            local ws_port
            ws_port=$(grep "^ENX_SESSION_GATEWAY_PUBLIC_WS_URL=" "$env_file" | sed 's/.*localhost://' | tr -d '\r')
            sed -i "s|^ENX_SESSION_GATEWAY_PUBLIC_WS_URL=ws://localhost:.*|ENX_SESSION_GATEWAY_PUBLIC_WS_URL=ws://${HOST_IP}:${ws_port}|" "$env_file"
            log_info "Patched ENX_SESSION_GATEWAY_PUBLIC_WS_URL: localhost -> ${HOST_IP}"
        fi
        if grep -q "^ENX_CORS_ALLOWED_ORIGINS=http://localhost:" "$env_file" 2>/dev/null; then
            local cors_val
            cors_val=$(grep "^ENX_CORS_ALLOWED_ORIGINS=" "$env_file" | sed 's/^ENX_CORS_ALLOWED_ORIGINS=//' | tr -d '\r')
            local new_cors="http://${HOST_IP}:3000,${cors_val}"
            if ! echo "$cors_val" | grep -q "${HOST_IP}"; then
                sed -i "s|^ENX_CORS_ALLOWED_ORIGINS=.*|ENX_CORS_ALLOWED_ORIGINS=${new_cors}|" "$env_file"
                log_info "Patched ENX_CORS_ALLOWED_ORIGINS: added ${HOST_IP}"
            fi
        fi
    fi
}

generate_env() {
    local env_file="${DEPLOY_DIR}/.env"

    if [ -f "$env_file" ]; then
        log_info ".env already exists, skipping generation."
        patch_env
        return
    fi

    log_info "Generating .env with auto-detected configuration..."

    local HOST_IP
    HOST_IP=$(detect_host_ip)

    local JWT_SECRET DEVICE_SECRET SESSION_SECRET MYSQL_PASS
    JWT_SECRET=$(gen_secret)
    DEVICE_SECRET=$(gen_secret)
    SESSION_SECRET=$(gen_secret)
    MYSQL_PASS=$(gen_secret)

    cat > "$env_file" <<EOF
# ============================================================
#  EnvNexus — Auto-generated configuration
#  Generated at: $(date '+%Y-%m-%d %H:%M:%S')
#  Host IP: ${HOST_IP}
# ============================================================

# ===== Database =====
ENX_DATABASE_DSN="root:${MYSQL_PASS}@tcp(mysql:3306)/envnexus?charset=utf8mb4&parseTime=True&loc=Local"
MYSQL_ROOT_PASSWORD=${MYSQL_PASS}
MYSQL_DATABASE=envnexus

# ===== Redis =====
ENX_REDIS_ADDR=redis:6379
ENX_REDIS_PASSWORD=

# ===== Object Storage (MinIO) =====
ENX_OBJECT_STORAGE_ENDPOINT=minio:9000
ENX_OBJECT_STORAGE_PUBLIC_ENDPOINT=${HOST_IP}:9000
ENX_OBJECT_STORAGE_BUCKET=envnexus
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin

# ===== Security & Secrets (auto-generated) =====
ENX_JWT_SECRET=${JWT_SECRET}
ENX_DEVICE_TOKEN_SECRET=${DEVICE_SECRET}
ENX_SESSION_TOKEN_SECRET=${SESSION_SECRET}

# ===== Platform API =====
ENX_PLATFORM_API_HOST=0.0.0.0
ENX_PLATFORM_API_PORT=8080
ENX_HTTP_PORT=8080
ENX_PLATFORM_API_PUBLIC_BASE_URL=http://${HOST_IP}:8080
ENX_CORS_ALLOWED_ORIGINS=http://${HOST_IP}:3000,http://localhost:3000
ENX_GATEWAY_URL=http://session-gateway:8081

# ===== Session Gateway =====
ENX_SESSION_GATEWAY_HOST=0.0.0.0
ENX_SESSION_GATEWAY_PORT=8081
ENX_SESSION_GATEWAY_PUBLIC_WS_URL=ws://${HOST_IP}:8081

# ===== Job Runner =====
ENX_HEALTH_PORT=8082
ENX_PLATFORM_URL=http://platform-api:8080
ENX_WS_URL=ws://session-gateway:8081/ws/v1/sessions

# ===== Console Web =====
ENX_CONSOLE_PORT=3000
ENX_CONSOLE_WEB_API_URL=http://platform-api:8080
EOF

    log_info "Generated .env with:"
    log_info "  Host IP:        ${HOST_IP}"
    log_info "  MySQL password: (auto-generated)"
    log_info "  JWT secret:     (auto-generated)"
    log_info "  Secrets:        (auto-generated)"
}

# ── .gitignore guard ─────────────────────────────────────────────────────────

ensure_gitignore() {
    local gitignore="${SCRIPT_DIR}/.gitignore"
    if [ -f "$gitignore" ]; then
        for pattern in "deploy/docker/volumes/" "deploy/docker/.env" "deploy/docker/.build-hashes/"; do
            if ! grep -q "^${pattern}$" "$gitignore" 2>/dev/null; then
                echo "$pattern" >> "$gitignore"
            fi
        done
    fi
}

# ── Print access URLs ────────────────────────────────────────────────────────

print_urls() {
    local HOST_IP
    HOST_IP=$(detect_host_ip)

    echo ""
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo -e "${GREEN}${BOLD}       EnvNexus is running! 🎉          ${NC}"
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo ""
    echo -e "  ${CYAN}Console Web:${NC}      http://${HOST_IP}:${ENX_CONSOLE_PORT:-3000}"
    echo -e "  ${CYAN}Platform API:${NC}     http://${HOST_IP}:${ENX_PLATFORM_API_PORT:-8080}"
    echo -e "  ${CYAN}Session Gateway:${NC}  ws://${HOST_IP}:${ENX_SESSION_GATEWAY_PORT:-8081}"
    echo ""
    echo -e "  ${YELLOW}View status:${NC}  ./deploy.sh status"
    echo -e "  ${YELLOW}View logs:${NC}    ./deploy.sh logs"
    echo -e "  ${YELLOW}Stop:${NC}         ./deploy.sh stop"
    echo ""
}

# ── Electron Shell image management ──────────────────────────────────────────
# The enx-electron-shell Docker image contains pre-installed npm deps, compiled
# TypeScript, and cached electron-builder tools. It only needs rebuilding when
# apps/agent-desktop/ source changes. This eliminates the #1 failure point
# (npm install / electron download) from routine agent-builder runs.

ensure_electron_shell() {
    local shell_hash
    shell_hash=$(compute_hash apps/agent-desktop)
    local needs_build=false

    # Check if image exists at all
    if ! docker image inspect enx-electron-shell &>/dev/null; then
        log_info "enx-electron-shell image not found — building for the first time..."
        needs_build=true
    elif has_changed "electron-shell" "$shell_hash"; then
        log_info "enx-electron-shell: ${YELLOW}apps/agent-desktop changed, rebuilding shell image...${NC}"
        needs_build=true
    else
        log_skip "enx-electron-shell: image up-to-date (apps/agent-desktop unchanged)"
    fi

    if $needs_build; then
        log_info "Building enx-electron-shell (one-time, includes npm install + electron download)..."
        log_info "  This may take 3-5 minutes on first run. Subsequent agent builds will be fast (~30s)."
        echo ""

        local exit_file
        exit_file=$(mktemp)
        echo "1" > "$exit_file"
        ( DOCKER_BUILDKIT=1 docker build \
            -f "${SCRIPT_DIR}/deploy/docker/electron-shell.Dockerfile" \
            -t enx-electron-shell \
            "${SCRIPT_DIR}" 2>&1
          echo $? > "$exit_file"
        ) | sed 's/^/  [electron-shell] /'
        local build_exit
        build_exit=$(cat "$exit_file" 2>/dev/null)
        rm -f "$exit_file"

        if [ "$build_exit" = "0" ]; then
            save_hash "electron-shell" "$shell_hash"
            log_info "enx-electron-shell: ${GREEN}image built and cached${NC}"
        else
            log_error "enx-electron-shell build failed! agent-builder will not work."
            log_error "  Fix the issue and retry with: ./deploy.sh agents-shell"
            return 1
        fi
        echo ""
    fi
}

cmd_build_electron_shell() {
    echo -e "${GREEN}${BOLD}  EnvNexus — Build Electron Shell Image  ${NC}"
    echo ""
    log_info "Force-building enx-electron-shell image..."
    log_info "  This pre-installs npm deps, compiles TypeScript, and caches electron-builder tools."
    echo ""

    cd "$SCRIPT_DIR"

    local exit_file
    exit_file=$(mktemp)
    echo "1" > "$exit_file"
    ( DOCKER_BUILDKIT=1 docker build \
        -f deploy/docker/electron-shell.Dockerfile \
        -t enx-electron-shell \
        . 2>&1
      echo $? > "$exit_file"
    ) | sed 's/^/  [electron-shell] /'
    local build_exit
    build_exit=$(cat "$exit_file" 2>/dev/null)
    rm -f "$exit_file"

    if [ "$build_exit" = "0" ]; then
        local shell_hash
        shell_hash=$(compute_hash apps/agent-desktop)
        save_hash "electron-shell" "$shell_hash"
        log_info "enx-electron-shell: ${GREEN}image built successfully${NC}"
        echo ""
        echo -e "  ${DIM}Now run './deploy.sh agents' or './deploy.sh start' to build agent packages.${NC}"
    else
        log_error "Build failed. Check the output above for errors."
    fi
}


build_agent_with_fallback() {
    local timeout_secs="${AGENT_BUILD_TIMEOUT:-300}"
    log_info "Attempting agent-builder in Docker (${timeout_secs}s timeout)..."

    # Run docker build with timeout, capture exit code via temp file (pipe to sed loses PIPESTATUS)
    local exit_file
    exit_file=$(mktemp)
    echo "1" > "$exit_file"
    ( timeout "${timeout_secs}" env DOCKER_BUILDKIT=1 docker compose --profile agent-build up --build agent-builder 2>&1
      echo $? > "$exit_file"
    ) | sed 's/^/  [agent-builder] /'
    local exit_code
    exit_code=$(cat "$exit_file" 2>/dev/null)
    rm -f "$exit_file"

    if [ "$exit_code" = "0" ]; then
        log_info "Docker agent build completed successfully."
        return 0
    fi

    if [ "$exit_code" = "124" ]; then
        log_warn "Docker agent build timed out (>${timeout_secs}s). Falling back to host build..."
        docker compose --profile agent-build stop agent-builder 2>/dev/null || true
    else
        log_warn "Docker agent build failed (exit $exit_code). Falling back to host build..."
    fi

    cd "$SCRIPT_DIR"
    log_info "Running 'make build-desktop' on host..."
    if make build-desktop 2>&1 | sed 's/^/  [host-build] /'; then
        log_info "Uploading host-built artifacts to MinIO..."

        # Detect the Compose network: inspect a running container to find the actual network
        local compose_network
        local minio_id
        minio_id=$(cd "$DEPLOY_DIR" && docker compose ps -q minio 2>/dev/null | head -1)
        if [ -n "$minio_id" ]; then
            compose_network=$(docker inspect "$minio_id" --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' 2>/dev/null | head -1)
        fi
        if [ -z "$compose_network" ]; then
            # Fallback: Compose project = directory name "docker" by default, default network = docker_default
            # If COMPOSE_PROJECT_NAME is set, use that instead.
            local proj_name="${COMPOSE_PROJECT_NAME:-docker}"
            compose_network="${proj_name}_default"
            log_warn "Could not detect Compose network from running containers, assuming '${compose_network}'"
        fi

        local upload_exit_file
        upload_exit_file=$(mktemp)
        echo "1" > "$upload_exit_file"
        ( docker run --rm --network "$compose_network" \
            -e "ENX_OBJECT_STORAGE_ENDPOINT=${ENX_OBJECT_STORAGE_ENDPOINT:-minio:9000}" \
            -e "ENX_OBJECT_STORAGE_BUCKET=${ENX_OBJECT_STORAGE_BUCKET:-envnexus}" \
            -e "MINIO_ROOT_USER=${MINIO_ROOT_USER:-minioadmin}" \
            -e "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD:-minioadmin}" \
            -e "FORCE_UPLOAD=${FORCE_AGENT_UPLOAD:-${FORCE_UPLOAD:-false}}" \
            -v "${SCRIPT_DIR}/bin/agents:/binaries:ro" \
            -v "${SCRIPT_DIR}/apps/agent-desktop/release:/installers:ro" \
            -v "${SCRIPT_DIR}/deploy/docker/upload-agents.sh:/upload-agents.sh:ro" \
            --entrypoint sh minio/mc:latest /upload-agents.sh 2>&1
          echo $? > "$upload_exit_file"
        ) | sed 's/^/  [host-upload] /'
        local upload_exit
        upload_exit=$(cat "$upload_exit_file" 2>/dev/null)
        rm -f "$upload_exit_file"
        return "$upload_exit"
    else
        log_error "Host build also failed."
        return 1
    fi
}

# ── Smart deploy: detect changes and only rebuild what's needed ──────────────


cmd_smart_deploy() {
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo -e "${GREEN}${BOLD}   EnvNexus — Smart Deploy              ${NC}"
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo ""

    generate_env
    ensure_gitignore

    set -a
    source "${DEPLOY_DIR}/.env"
    set +a

    cd "$DEPLOY_DIR"

    local go_ws_hash
    go_ws_hash=$(compute_go_workspace_hash)

    # Define service -> source directory mapping
    # Each service hash = service source hash + go workspace hash (for Go services)
    local services_to_build=""
    local services_to_skip=""

    # ── Check each service ──

    # 1. platform-api
    local pa_hash
    pa_hash=$(echo "$(compute_hash services/platform-api)${go_ws_hash}" | sha256sum | awk '{print $1}')
    if has_changed "platform-api" "$pa_hash"; then
        services_to_build="${services_to_build} platform-api"
        log_info "platform-api: ${YELLOW}source changed, will rebuild${NC}"
    else
        services_to_skip="${services_to_skip} platform-api"
        log_skip "platform-api: no changes detected"
    fi

    # 2. session-gateway
    local sg_hash
    sg_hash=$(echo "$(compute_hash services/session-gateway)${go_ws_hash}" | sha256sum | awk '{print $1}')
    if has_changed "session-gateway" "$sg_hash"; then
        services_to_build="${services_to_build} session-gateway"
        log_info "session-gateway: ${YELLOW}source changed, will rebuild${NC}"
    else
        services_to_skip="${services_to_skip} session-gateway"
        log_skip "session-gateway: no changes detected"
    fi

    # 3. job-runner
    local jr_hash
    jr_hash=$(echo "$(compute_hash services/job-runner)${go_ws_hash}" | sha256sum | awk '{print $1}')
    if has_changed "job-runner" "$jr_hash"; then
        services_to_build="${services_to_build} job-runner"
        log_info "job-runner: ${YELLOW}source changed, will rebuild${NC}"
    else
        services_to_skip="${services_to_skip} job-runner"
        log_skip "job-runner: no changes detected"
    fi

    # 4. console-web
    local cw_hash
    cw_hash=$(compute_hash apps/console-web)
    if has_changed "console-web" "$cw_hash"; then
        services_to_build="${services_to_build} console-web"
        log_info "console-web: ${YELLOW}source changed, will rebuild${NC}"
    else
        services_to_skip="${services_to_skip} console-web"
        log_skip "console-web: no changes detected"
    fi

    # 5. agent-builder (agent-core + agent-desktop)
    #    Electron shell image is built separately and cached as enx-electron-shell.
    #    agent-builder only needs to cross-compile Go + package installers (fast).
    local ab_hash
    ab_hash=$(echo "$(compute_hash apps/agent-core)$(compute_hash apps/agent-desktop)" | sha256sum | awk '{print $1}')
    if has_changed "agent-builder" "$ab_hash"; then
        services_to_build="${services_to_build} agent-builder"
        log_info "agent-builder: ${YELLOW}source changed, will rebuild & upload${NC}"
    else
        services_to_skip="${services_to_skip} agent-builder"
        log_skip "agent-builder: no changes detected (agent-core + agent-desktop unchanged)"
    fi

    echo ""

    # ── Ensure Electron shell image exists (prerequisite for agent-builder) ──
    if echo "$services_to_build" | grep -q "agent-builder"; then
        ensure_electron_shell
    fi

    # ── Infrastructure services (always ensure running, never rebuild) ──
    log_info "Ensuring infrastructure services are running (mysql, redis, minio)..."
    docker compose up -d mysql redis minio minio-init

    # ── Build & deploy changed services ──
    local all_to_build="$services_to_build"
    if [ -n "$services_to_build" ]; then
        services_to_build=$(echo "$services_to_build" | xargs)

        echo ""
        log_info "Rebuilding: ${BOLD}${services_to_build}${NC}"
        echo ""

        local agent_needs_build=false
        local other_services=""
        if echo "$services_to_build" | grep -q "agent-builder"; then
            agent_needs_build=true
            other_services=$(echo "$services_to_build" | sed 's/agent-builder//g' | xargs)
        else
            other_services="$services_to_build"
        fi

        if $agent_needs_build && [ -n "$other_services" ]; then
            # ── PARALLEL: agent-builder + other services build simultaneously ──
            log_info "Parallel build: agent-builder ║ ${other_services}"
            log_info "  Track A: agent-builder (Go cross-compile + Electron + upload to MinIO)"
            log_info "  Track B: ${other_services} (image builds)"
            echo ""

            # Track A: agent-builder in background (build + run upload) with fallback
            # Disable 'set -e' inside the subshell so a failure doesn't kill the subshell silently before returning the exit code
            (
                set +e
                build_agent_with_fallback
            ) &
            local agent_pid=$!

            # Track B: other service images build + start (parallel via Compose)
            docker compose up -d --build $other_services 2>&1 \
                | sed 's/^/  [services]      /'

            # Wait for agent-builder to finish uploading
            log_info "Waiting for agent-builder upload to complete..."
            if wait $agent_pid 2>/dev/null || [ $? -eq 0 ]; then
                save_hash "agent-builder" "$ab_hash"
                log_info "agent-builder: ${GREEN}build + upload complete${NC}"
            else
                log_warn "agent-builder: build or upload failed (other services unaffected)"
            fi

        elif $agent_needs_build; then
            # Only agent-builder needs rebuild, no other services
            log_info "Building agent-builder (Go cross-compile + Electron installers + upload)..."
            if build_agent_with_fallback; then
                save_hash "agent-builder" "$ab_hash"
            else
                log_warn "agent-builder: build failed, hash NOT saved (will retry next deploy)"
            fi

        elif [ -n "$other_services" ]; then
            # Only app services need rebuild (agent unchanged)
            docker compose up -d --build $other_services
        fi

        # Save hashes for successfully built app services
        # Use here-string to avoid pipe subshell (pipe | while runs in subshell in bash)
        local svc
        for svc in $other_services; do
            [ -z "$svc" ] && continue
            case "$svc" in
                platform-api)     save_hash "platform-api" "$pa_hash" ;;
                session-gateway)  save_hash "session-gateway" "$sg_hash" ;;
                job-runner)       save_hash "job-runner" "$jr_hash" ;;
                console-web)      save_hash "console-web" "$cw_hash" ;;
            esac
        done
    else
        log_info "No service code changes detected."
    fi

    # ── Ensure all services are running (start any that are stopped) ──
    # build_agent_with_fallback may cd to SCRIPT_DIR (host build path); compose file lives in DEPLOY_DIR
    cd "$DEPLOY_DIR"
    log_info "Ensuring all services are up..."
    docker compose up -d

    echo ""
    local built_count=$(echo "$all_to_build" | xargs | wc -w | xargs)
    local skipped_count=$(echo "$services_to_skip" | xargs | wc -w | xargs)
    echo -e "${GREEN}${BOLD}════════════════════════════════════════${NC}"
    echo -e "${GREEN}  Smart Deploy Complete${NC}"
    echo -e "  ${CYAN}Rebuilt:${NC} ${built_count} service(s)   ${DIM}Skipped:${NC} ${skipped_count} service(s)"
    echo -e "${GREEN}${BOLD}════════════════════════════════════════${NC}"

    print_urls
}

# ── Full rebuild (force all) ─────────────────────────────────────────────────

cmd_deploy_full() {
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo -e "${GREEN}${BOLD}   EnvNexus — Full Rebuild (no cache)   ${NC}"
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo ""

    generate_env
    ensure_gitignore

    set -a
    source "${DEPLOY_DIR}/.env"
    set +a

    cd "$DEPLOY_DIR"

    # Clear all build hashes to force rebuild
    rm -f "${HASH_DIR}"/*.hash

    # Ensure electron shell image exists (required for agent-builder)
    ensure_electron_shell

    log_info "Starting infrastructure..."
    docker compose up -d mysql redis minio minio-init

    log_info "Full parallel rebuild: agent-builder ║ all app services"
    echo ""

    # Track A: agent-builder (background — build + upload to MinIO) with fallback
    # Disable 'set -e' inside the subshell so a failure doesn't kill the subshell silently before returning the exit code
    (
        set +e
        build_agent_with_fallback
    ) &
    local agent_pid=$!

    # Track B: all app services (parallel image builds via Compose)
    docker compose up -d --build platform-api session-gateway job-runner console-web 2>&1 \
        | sed 's/^/  [services]      /'

    # Wait for agent-builder upload to finish
    local agent_build_ok=false
    log_info "Waiting for agent-builder upload to complete..."
    if wait $agent_pid 2>/dev/null || [ $? -eq 0 ]; then
        agent_build_ok=true
        log_info "agent-builder: ${GREEN}build + upload complete${NC}"
    else
        log_warn "agent-builder: build or upload had issues (check logs)"
    fi

    # Ensure everything is up
    docker compose up -d

    # Save hashes — only for components that succeeded
    local go_ws_hash
    go_ws_hash=$(compute_go_workspace_hash)
    save_hash "platform-api" "$(echo "$(compute_hash services/platform-api)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    save_hash "session-gateway" "$(echo "$(compute_hash services/session-gateway)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    save_hash "job-runner" "$(echo "$(compute_hash services/job-runner)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    save_hash "console-web" "$(compute_hash apps/console-web)"
    if $agent_build_ok; then
        save_hash "agent-builder" "$(echo "$(compute_hash apps/agent-core)$(compute_hash apps/agent-desktop)" | sha256sum | awk '{print $1}')"
    fi

    print_urls
}

cmd_deploy_web() {
    echo -e "${GREEN}${BOLD}  EnvNexus — Rebuild Console Web only   ${NC}"
    cd "$DEPLOY_DIR"
    docker compose up -d --build console-web
    save_hash "console-web" "$(compute_hash apps/console-web)"
    log_info "Console Web redeployed."
}

cmd_deploy_api() {
    echo -e "${GREEN}${BOLD}  EnvNexus — Rebuild Backend only       ${NC}"
    cd "$DEPLOY_DIR"
    docker compose up -d --build platform-api session-gateway job-runner
    local go_ws_hash
    go_ws_hash=$(compute_go_workspace_hash)
    save_hash "platform-api" "$(echo "$(compute_hash services/platform-api)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    save_hash "session-gateway" "$(echo "$(compute_hash services/session-gateway)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    save_hash "job-runner" "$(echo "$(compute_hash services/job-runner)${go_ws_hash}" | sha256sum | awk '{print $1}')"
    log_info "Backend services redeployed."
}

cmd_stop() {
    log_warn "Stopping all EnvNexus services..."
    cd "$DEPLOY_DIR"
    docker compose down
    log_info "All services stopped. Data is preserved in volumes/."
}

cmd_restart() {
    log_warn "Restarting all EnvNexus services..."
    cd "$DEPLOY_DIR"

    set -a
    source "${DEPLOY_DIR}/.env"
    set +a

    docker compose restart
    if [ $? -eq 0 ]; then
        print_urls
    else
        log_error "Restart failed."
    fi
}

cmd_status() {
    cd "$DEPLOY_DIR"
    docker compose ps

    echo ""
    echo -e "${BOLD}Build hash status:${NC}"
    if [ -d "$HASH_DIR" ] && ls "$HASH_DIR"/*.hash &>/dev/null; then
        for f in "$HASH_DIR"/*.hash; do
            local svc=$(basename "$f" .hash)
            local hash=$(cat "$f" | head -c 12)
            echo -e "  ${svc}: ${DIM}${hash}...${NC}"
        done
    else
        echo "  (no build hashes recorded yet — run 'deploy.sh start' first)"
    fi
}

cmd_logs() {
    cd "$DEPLOY_DIR"
    local service="${1:-}"
    if [ -n "$service" ]; then
        docker compose logs -f "$service"
    else
        docker compose logs -f
    fi
}

cmd_reset() {
    log_warn "This will DELETE all data (database, redis, minio) and regenerate secrets."
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$DEPLOY_DIR"
        docker compose down -v 2>/dev/null || true
        rm -rf "${DEPLOY_DIR}/volumes" "${DEPLOY_DIR}/.env" "${HASH_DIR}"
        log_info "All data and config removed. Run './deploy.sh start' to redeploy."
    else
        log_info "Cancelled."
    fi
}

# ── Main ─────────────────────────────────────────────────────────────────────

check_env

case "${1:-}" in
    start|deploy|up)
        cmd_smart_deploy
        ;;
    full|rebuild)
        cmd_deploy_full
        ;;
    web)
        cmd_deploy_web
        ;;
    api|backend)
        cmd_deploy_api
        ;;
    stop|down)
        cmd_stop
        ;;
    restart)
        cmd_restart
        ;;
    status|ps)
        cmd_status
        ;;
    logs)
        cmd_logs "${2:-}"
        ;;
    reset)
        cmd_reset
        ;;
    agents)
        echo -e "${GREEN}${BOLD}  EnvNexus — Force rebuild agent packages  ${NC}"
        ensure_electron_shell
        cd "$DEPLOY_DIR"
        set -a; source "${DEPLOY_DIR}/.env"; set +a
        if FORCE_AGENT_UPLOAD=true build_agent_with_fallback; then
            save_hash "agent-builder" "$(echo "$(compute_hash apps/agent-core)$(compute_hash apps/agent-desktop)" | sha256sum | awk '{print $1}')"
            log_info "Agent packages rebuilt and uploaded to MinIO."
        else
            log_error "Agent build failed. Hash NOT saved — will retry next time."
        fi
        ;;
    agents-shell)
        cmd_build_electron_shell
        ;;
    *)
        echo -e "${BOLD}EnvNexus — Smart Deployment Manager${NC}"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo -e "${BOLD}Commands:${NC}"
        echo "  start        Smart deploy — only rebuild services with code changes"
        echo "  full         Force rebuild all services (ignore cache)"
        echo "  web          Rebuild and redeploy console-web only"
        echo "  api          Rebuild and redeploy backend services only"
        echo "  agents       Force rebuild and re-upload agent packages (Go + Electron)"
        echo "  agents-shell Build/rebuild the Electron shell image (one-time setup)"
        echo "  stop         Stop all services (data preserved)"
        echo "  restart      Restart all services without rebuild"
        echo "  status       Show running services and build hash status"
        echo "  logs         Tail logs (optionally: logs <service>)"
        echo "  reset        Delete all data and config, start fresh"
        echo ""
        echo -e "${BOLD}Agent Build Architecture:${NC}"
        echo "  The agent build is split into two layers for speed and reliability:"
        echo "  1. Electron Shell (enx-electron-shell image) — built once, includes npm deps"
        echo "     and electron-builder cache. Rebuild with: ./deploy.sh agents-shell"
        echo "  2. Agent Core (Go binary) — fast cross-compile (~30s), injected into shell"
        echo ""
        echo -e "${DIM}Smart deploy computes content hashes of each service's source code."
        echo -e "If the hash matches the last successful build, the service is skipped.${NC}"
        exit 1
        ;;
esac
