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

for f in /agents/enx-agent-*; do
    name=$(basename "$f")
    remote="enx/${BUCKET}/base-packages/${name}"
    local_size=$(stat -c%s "$f" 2>/dev/null || stat -f%z "$f" 2>/dev/null)

    if [ "$FORCE" != "true" ]; then
        remote_size=$(mc stat "$remote" 2>/dev/null | grep "Size" | awk '{print $3}' || echo "")
        if [ -n "$remote_size" ] && [ "$remote_size" = "$local_size" ]; then
            echo "  ⏭ ${name} (already exists, same size ${local_size}B)"
            skipped=$((skipped + 1))
            continue
        fi
    fi

    echo "  Uploading ${name} (${local_size}B)..."
    mc cp "$f" "$remote"
    echo "  ✓ ${name}"
    uploaded=$((uploaded + 1))
done

echo ""
echo "Done: ${uploaded} uploaded, ${skipped} skipped (already up-to-date)"

if [ "$uploaded" -gt 0 ] || [ "$skipped" -eq 0 ]; then
    echo ""
    echo "Base packages in MinIO:"
    mc ls "enx/${BUCKET}/base-packages/"
fi
