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
    if [ -z "$ip" ] && command -v ipconfig &>/dev/null; then
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
    if [ -d "${SCRIPT_DIR}/${dir}" ]; then
        find "${SCRIPT_DIR}/${dir}" -type f \
            -not -path '*/node_modules/*' \
            -not -path '*/.next/*' \
            -not -path '*/dist/*' \
            -not -path '*/release/*' \
            -not -path '*/__pycache__/*' \
            -print0 2>/dev/null | sort -z | xargs -0 sha256sum 2>/dev/null | sha256sum | awk '{print $1}'
    else
        echo "missing"
    fi
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

    # ── Infrastructure services (always ensure running, never rebuild) ──
    log_info "Ensuring infrastructure services are running (mysql, redis, minio)..."
    docker compose up -d mysql redis minio minio-init

    # ── Build & deploy changed services ──
    local all_to_build="$services_to_build"
    if [ -n "$services_to_build" ]; then
        # Trim leading space
        services_to_build=$(echo "$services_to_build" | xargs)

        echo ""
        log_info "Rebuilding: ${BOLD}${services_to_build}${NC}"
        echo ""

        # If agent-builder needs rebuild, do it first (uploads base packages to MinIO)
        if echo "$services_to_build" | grep -q "agent-builder"; then
            log_info "Building agent-builder (Go cross-compile + Electron installers + upload)..."
            docker compose --profile agent-build up --build agent-builder
            save_hash "agent-builder" "$ab_hash"
            # Remove agent-builder from the remaining list
            services_to_build=$(echo "$services_to_build" | sed 's/agent-builder//g' | xargs)
        fi

        # Build remaining services in parallel
        if [ -n "$services_to_build" ]; then
            docker compose up -d --build $services_to_build
        fi

        # Save hashes for successfully built services
        echo "$services_to_build" | tr ' ' '\n' | while read -r svc; do
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

    log_info "Building and starting all services (including agent-builder)..."
    docker compose --profile agent-build up -d --build

    if [ $? -eq 0 ]; then
        # Save all hashes after successful full build
        local go_ws_hash
        go_ws_hash=$(compute_go_workspace_hash)
        save_hash "platform-api" "$(echo "$(compute_hash services/platform-api)${go_ws_hash}" | sha256sum | awk '{print $1}')"
        save_hash "session-gateway" "$(echo "$(compute_hash services/session-gateway)${go_ws_hash}" | sha256sum | awk '{print $1}')"
        save_hash "job-runner" "$(echo "$(compute_hash services/job-runner)${go_ws_hash}" | sha256sum | awk '{print $1}')"
        save_hash "console-web" "$(compute_hash apps/console-web)"
        save_hash "agent-builder" "$(echo "$(compute_hash apps/agent-core)$(compute_hash apps/agent-desktop)" | sha256sum | awk '{print $1}')"
        print_urls
    else
        log_error "Deployment failed. Check logs: ./deploy.sh logs"
        exit 1
    fi
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
        echo -e "${GREEN}${BOLD}  EnvNexus — Force rebuild agent binaries  ${NC}"
        cd "$DEPLOY_DIR"
        FORCE_AGENT_UPLOAD=true docker compose --profile agent-build up --build --force-recreate agent-builder
        save_hash "agent-builder" "$(echo "$(compute_hash apps/agent-core)$(compute_hash apps/agent-desktop)" | sha256sum | awk '{print $1}')"
        log_info "Agent binaries rebuilt and uploaded to MinIO."
        ;;
    *)
        echo -e "${BOLD}EnvNexus — Smart Deployment Manager${NC}"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo -e "${BOLD}Commands:${NC}"
        echo "  start     Smart deploy — only rebuild services with code changes"
        echo "  full      Force rebuild all services (ignore cache)"
        echo "  web       Rebuild and redeploy console-web only"
        echo "  api       Rebuild and redeploy backend services only"
        echo "  agents    Force rebuild and re-upload agent base packages"
        echo "  stop      Stop all services (data preserved)"
        echo "  restart   Restart all services without rebuild"
        echo "  status    Show running services and build hash status"
        echo "  logs      Tail logs (optionally: logs <service>)"
        echo "  reset     Delete all data and config, start fresh"
        echo ""
        echo -e "${DIM}Smart deploy computes content hashes of each service's source code."
        echo -e "If the hash matches the last successful build, the service is skipped.${NC}"
        exit 1
        ;;
esac
