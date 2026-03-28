#!/bin/bash
set -euo pipefail

# EnvNexus — One-click local deployment script
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="${SCRIPT_DIR}/deploy/docker"

# ── Helpers ──────────────────────────────────────────────────────────────────

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

gen_secret() {
    # Generate a 32-char random hex string; works on Linux/macOS
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
    # Linux
    if command -v hostname &>/dev/null && hostname -I &>/dev/null 2>&1; then
        ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    # macOS
    if [ -z "$ip" ] && command -v ipconfig &>/dev/null; then
        ip=$(ipconfig getifaddr en0 2>/dev/null || true)
    fi
    # Fallback
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

# ── .env generation ──────────────────────────────────────────────────────────

generate_env() {
    local env_file="${DEPLOY_DIR}/.env"

    if [ -f "$env_file" ]; then
        log_info ".env already exists, skipping generation."
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
        for pattern in "deploy/docker/volumes/" "deploy/docker/.env"; do
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

# ── Commands ─────────────────────────────────────────────────────────────────

cmd_deploy() {
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo -e "${GREEN}${BOLD}   EnvNexus — One-click Local Deploy    ${NC}"
    echo -e "${GREEN}${BOLD}========================================${NC}"
    echo ""

    generate_env
    ensure_gitignore

    # Source .env so we can use port variables in print_urls
    set -a
    source "${DEPLOY_DIR}/.env"
    set +a

    cd "$DEPLOY_DIR"
    log_info "Building and starting all services..."
    docker compose up -d --build

    if [ $? -eq 0 ]; then
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
    log_info "Console Web redeployed."
}

cmd_deploy_api() {
    echo -e "${GREEN}${BOLD}  EnvNexus — Rebuild Backend only       ${NC}"
    cd "$DEPLOY_DIR"
    docker compose up -d --build platform-api session-gateway job-runner
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
        rm -rf "${DEPLOY_DIR}/volumes" "${DEPLOY_DIR}/.env"
        log_info "All data and config removed. Run './deploy.sh start' to redeploy."
    else
        log_info "Cancelled."
    fi
}

# ── Main ─────────────────────────────────────────────────────────────────────

check_env

case "${1:-}" in
    start|deploy|up)
        cmd_deploy
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
    *)
        echo -e "${BOLD}EnvNexus — Deployment Manager${NC}"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo -e "${BOLD}Commands:${NC}"
        echo "  start     Deploy and start all services (one-click)"
        echo "  web       Rebuild and redeploy console-web only"
        echo "  api       Rebuild and redeploy backend services only"
        echo "  stop      Stop all services (data preserved)"
        echo "  restart   Restart all services without rebuild"
        echo "  status    Show running services"
        echo "  logs      Tail logs (optionally: logs <service>)"
        echo "  reset     Delete all data and config, start fresh"
        exit 1
        ;;
esac
