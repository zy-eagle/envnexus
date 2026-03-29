#!/bin/sh
set -e

ENDPOINT="${ENX_OBJECT_STORAGE_ENDPOINT:-minio:9000}"
BUCKET="${ENX_OBJECT_STORAGE_BUCKET:-envnexus}"
ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
FORCE="${FORCE_UPLOAD:-false}"

echo "Configuring MinIO alias (endpoint: ${ENDPOINT})..."
mc alias set enx "http://${ENDPOINT}" "${ACCESS_KEY}" "${SECRET_KEY}" --api S3v4

echo "Ensuring bucket exists..."
mc mb --ignore-existing "enx/${BUCKET}"

uploaded=0
skipped=0

upload_file() {
    local file="$1"
    local name=$(basename "$file")
    local remote="enx/${BUCKET}/base-packages/${name}"
    local local_size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null)

    if [ "$FORCE" != "true" ]; then
        # Check if remote file exists and has the same size
        remote_info=$(mc stat "$remote" 2>/dev/null || echo "")
        if [ -n "$remote_info" ]; then
            remote_size=$(echo "$remote_info" | grep "Size" | awk '{print $3}' || echo "")
            if [ -n "$remote_size" ] && [ "$remote_size" = "$local_size" ]; then
                # Same size — compute local hash for deeper check
                local_hash=$(md5sum "$file" 2>/dev/null | awk '{print $1}' || md5 -q "$file" 2>/dev/null || echo "")
                remote_etag=$(echo "$remote_info" | grep -i "ETag" | awk '{print $NF}' | tr -d '"' || echo "")
                if [ -n "$local_hash" ] && [ -n "$remote_etag" ] && [ "$local_hash" = "$remote_etag" ]; then
                    echo "  ⏭ ${name} (identical, skipped)"
                    skipped=$((skipped + 1))
                    return
                fi
                # If we can't compare hashes, size match is good enough
                if [ -z "$local_hash" ] || [ -z "$remote_etag" ]; then
                    echo "  ⏭ ${name} (same size, skipped)"
                    skipped=$((skipped + 1))
                    return
                fi
            fi
        fi
    fi

    echo "  ⬆ Uploading ${name} ($(du -h "$file" | cut -f1))..."
    mc cp "$file" "$remote"
    echo "  ✓ ${name}"
    uploaded=$((uploaded + 1))
}

# ── Priority 1: Desktop Installers (NSIS .exe, AppImage, DMG) ──────────────
echo ""
echo "=== Desktop Installers (GUI + Agent Core bundled) ==="
installer_count=0
for f in /installers/*; do
    if [ -f "$f" ]; then
        upload_file "$f"
        installer_count=$((installer_count + 1))
    fi
done
if [ "$installer_count" -eq 0 ]; then
    echo "  (none found — electron-builder may have failed, falling back to raw binaries)"
fi

# ── Priority 2: Raw Agent Core binaries (fallback / headless servers) ──────
echo ""
echo "=== Raw Agent Core Binaries (CLI-only, for servers) ==="
for f in /binaries/enx-agent-*; do
    [ -f "$f" ] && upload_file "$f"
done

echo ""
echo "════════════════════════════════════════════════════"
echo "  Done: ${uploaded} uploaded, ${skipped} skipped"
echo "════════════════════════════════════════════════════"

echo ""
echo "Base packages in MinIO:"
mc ls "enx/${BUCKET}/base-packages/" 2>/dev/null || echo "(empty)"
