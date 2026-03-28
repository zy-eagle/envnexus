#!/bin/bash
set -euo pipefail

# Cross-compile enx-agent for all supported platform/arch combinations
# and upload the binaries to MinIO under base-packages/

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
AGENT_SRC="${PROJECT_ROOT}/apps/agent-core"
OUTPUT_DIR="${PROJECT_ROOT}/bin/agents"

PLATFORMS=("linux" "windows" "darwin")
ARCHES=("amd64" "arm64")

log_info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ── Phase 1: Cross-compile ──────────────────────────────────────────────────

build_agents() {
    log_info "Cross-compiling enx-agent for all platforms..."
    mkdir -p "${OUTPUT_DIR}"

    for os in "${PLATFORMS[@]}"; do
        for arch in "${ARCHES[@]}"; do
            local ext=""
            if [ "$os" = "windows" ]; then
                ext=".exe"
            fi
            local binary_name="enx-agent-${os}-${arch}${ext}"
            log_info "  Building ${CYAN}${binary_name}${NC} ..."

            (cd "${AGENT_SRC}" && \
                CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
                go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${binary_name}" ./cmd/enx-agent)

            local size
            size=$(du -h "${OUTPUT_DIR}/${binary_name}" | cut -f1)
            log_info "  ✓ ${binary_name} (${size})"
        done
    done

    log_info "All binaries built in ${OUTPUT_DIR}/"
}

# ── Phase 2: Upload to MinIO ────────────────────────────────────────────────

upload_to_minio() {
    local endpoint="${ENX_OBJECT_STORAGE_ENDPOINT:-}"
    local bucket="${ENX_OBJECT_STORAGE_BUCKET:-envnexus}"
    local access_key="${MINIO_ROOT_USER:-minioadmin}"
    local secret_key="${MINIO_ROOT_PASSWORD:-minioadmin}"

    if [ -z "$endpoint" ]; then
        log_warn "ENX_OBJECT_STORAGE_ENDPOINT not set. Skipping MinIO upload."
        log_warn "Set it to upload, e.g.: ENX_OBJECT_STORAGE_ENDPOINT=localhost:9000"
        return 0
    fi

    if ! command -v mc &>/dev/null; then
        log_error "'mc' (MinIO Client) not found. Install it: https://min.io/docs/minio/linux/reference/minio-mc.html"
        log_warn "Binaries are available locally at ${OUTPUT_DIR}/ — upload manually."
        return 1
    fi

    log_info "Configuring MinIO alias..."
    mc alias set enx-build "http://${endpoint}" "${access_key}" "${secret_key}" --api S3v4 2>/dev/null

    log_info "Uploading base packages to MinIO (${endpoint}/${bucket}/base-packages/)..."

    for os in "${PLATFORMS[@]}"; do
        for arch in "${ARCHES[@]}"; do
            local ext=""
            if [ "$os" = "windows" ]; then
                ext=".exe"
            fi
            local binary_name="enx-agent-${os}-${arch}${ext}"
            local local_path="${OUTPUT_DIR}/${binary_name}"
            local remote_path="enx-build/${bucket}/base-packages/${binary_name}"

            if [ ! -f "${local_path}" ]; then
                log_warn "  ⚠ ${binary_name} not found, skipping"
                continue
            fi

            log_info "  Uploading ${CYAN}${binary_name}${NC} ..."
            mc cp "${local_path}" "${remote_path}"
            log_info "  ✓ ${binary_name} uploaded"
        done
    done

    log_info "All base packages uploaded to MinIO."
    echo ""
    log_info "Verifying uploads..."
    mc ls "enx-build/${bucket}/base-packages/" 2>/dev/null || true
}

# ── Main ────────────────────────────────────────────────────────────────────

echo -e "${BOLD}========================================${NC}"
echo -e "${BOLD}  EnvNexus Agent — Build & Upload       ${NC}"
echo -e "${BOLD}========================================${NC}"
echo ""

case "${1:-all}" in
    build)
        build_agents
        ;;
    upload)
        upload_to_minio
        ;;
    all)
        build_agents
        echo ""
        upload_to_minio
        ;;
    *)
        echo "Usage: $0 [build|upload|all]"
        echo ""
        echo "  build   — Cross-compile agent binaries only"
        echo "  upload  — Upload pre-built binaries to MinIO only"
        echo "  all     — Build + upload (default)"
        echo ""
        echo "Environment variables:"
        echo "  ENX_OBJECT_STORAGE_ENDPOINT  MinIO endpoint (e.g. localhost:9000)"
        echo "  ENX_OBJECT_STORAGE_BUCKET    Bucket name (default: envnexus)"
        echo "  MINIO_ROOT_USER              Access key (default: minioadmin)"
        echo "  MINIO_ROOT_PASSWORD          Secret key (default: minioadmin)"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}${BOLD}Done!${NC}"
